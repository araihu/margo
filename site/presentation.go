package site

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/ssg"
)

// LayoutProfiles names the trusted frame profiles available to configured
// pages. Profile names are semantic site-owned values; pages never select a
// frame distribution directly.
type LayoutProfiles struct {
	Default  string                   `yaml:"default"`
	Profiles map[string]LayoutProfile `yaml:"profiles"`
}

// LayoutProfile selects one concrete frame for a semantic page layout.
type LayoutProfile struct {
	Frame LayoutSelection `yaml:"frame"`
}

// FamilyConfig describes one publication family and its source subtree.
type FamilyConfig struct {
	ID       string `yaml:"id"`
	Label    string `yaml:"label"`
	Source   string `yaml:"source"`
	Overview string `yaml:"overview"`
	Layout   string `yaml:"layout"`
}

// PagePresentation is the resolved frame identity used by a configured page.
// Task 4 attaches these values to configured pages; keeping this model pure
// lets family/layout resolution be tested without rendering or staging.
type PagePresentation struct {
	FamilyID   string
	LayoutName string
	FrameName  string
	Frame      ssg.Frame
	Schema     ssg.FrameSchema
	Values     ssg.Values
	SchemaHash string
}

func configUsesLayoutProfiles(config Config) bool {
	return config.layoutsPresent || config.Layouts.Default != "" || len(config.Layouts.Profiles) > 0 || len(config.Navigation.Families) > 0
}

// prepareFramePresentations resolves every configured frame before page
// preflight. Profile names are traversed in sorted order so map-backed config
// cannot influence the prepared identity or any future staged output.
func prepareFramePresentations(config Config) (map[string]PagePresentation, error) {
	names := make([]string, 0, len(config.Layouts.Profiles))
	for name := range config.Layouts.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	presentations := make(map[string]PagePresentation, len(names))
	for _, name := range names {
		profile := config.Layouts.Profiles[name]
		frameName := profile.Frame.Builtin
		frame, err := ssg.BuiltinFrame(frameName)
		if err != nil {
			return nil, newPresentationDiagnostic("site.layout_unknown", err.Error(), "Select a supported builtin frame.", "/layouts/profiles/"+name+"/frame/builtin")
		}
		schema, err := frame.Schema(ssg.FrameContext{
			Locale: localeOrDefault(config.Locales.Default), Direction: localeDirection(config.Locales.Default),
			Profile: ssg.DocsProfile, Root: true, InstanceID: "root",
			Theme: ssg.ThemeContext{Name: config.Theme.Name, ColorMode: config.Theme.ColorMode},
		})
		if err != nil {
			return nil, err
		}
		if err := ssg.ValidateFrameSchema(schema, ssg.DocsProfile); err != nil {
			return nil, newPresentationDiagnostic("site.layout_invalid", err.Error(), "Select a navigation-capable root frame.", "/layouts/profiles/"+name+"/frame/builtin")
		}
		values, err := ssg.ResolveFrameValues(schema, ssg.Values(profile.Frame.Values))
		if err != nil {
			return nil, newPresentationDiagnostic("site.layout_values_invalid", err.Error(), "Use only option paths and values published by the selected frame.", "/layouts/profiles/"+name+"/frame/values")
		}
		hash, err := ssg.SchemaHashForValues(schema, values)
		if err != nil {
			return nil, err
		}
		presentations[name] = PagePresentation{LayoutName: name, FrameName: frameName, Frame: frame, Schema: schema, Values: values, SchemaHash: hash}
	}
	return presentations, nil
}

func localeOrDefault(locale string) string {
	if strings.TrimSpace(locale) == "" {
		return "en"
	}
	return locale
}

// profileLayoutIdentity records all semantic profiles, concrete frames, and
// resolved schema/value hashes. This keeps route/site identity distinct when
// two profiles use different frames or structural values.
func profileLayoutIdentity(presentations map[string]PagePresentation) (string, string) {
	names := make([]string, 0, len(presentations))
	for name := range presentations {
		names = append(names, name)
	}
	sort.Strings(names)
	identity := make([]string, 0, len(names))
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.ssg.layout-profiles/v1\x00"))
	for _, name := range names {
		presentation := presentations[name]
		identity = append(identity, name+"="+presentation.FrameName)
		for _, value := range []string{name, presentation.LayoutName, presentation.FrameName, presentation.SchemaHash} {
			_, _ = hash.Write([]byte(value))
			_, _ = hash.Write([]byte{0})
		}
	}
	return "profiles:" + strings.Join(identity, ","), hex.EncodeToString(hash.Sum(nil))
}

// resolveFamily returns the declaration with the most-specific segment-aware
// source prefix. Locale directories are removed before matching, so one
// family declaration serves every configured locale.
func resolveFamily(source string, locales LocaleConfig, families []FamilyConfig) (FamilyConfig, error) {
	originalSource := source
	_, source = sourceLocaleForPresentation(source, locales)
	if cleaned, ok := validSourcePath(source); ok {
		source = cleaned
	} else {
		source = path.Clean(source)
	}

	best := -1
	bestDepth := -1
	for index, family := range families {
		root := normalizeFamilyRootForMatch(family.Source)
		if !familyRootMatches(root, source) {
			continue
		}
		depth := familyRootDepth(root)
		if depth > bestDepth {
			best = index
			bestDepth = depth
		}
	}
	if best < 0 {
		return FamilyConfig{}, presentationSourceDiagnostic(newPresentationDiagnostic(
			"site.family_unresolved",
			fmt.Sprintf("source %q does not belong to a configured navigation family", source),
			fmt.Sprintf("Add a family whose source prefix contains %q.", source),
			"/navigation/families",
		), originalSource)
	}
	return families[best], nil
}

// resolveLayout applies page, family, then site-default precedence. Empty
// optional page/family values fall through; once a name is selected it must be
// present in the profile map.
func resolveLayout(metadata margo.Metadata, family FamilyConfig, layouts LayoutProfiles) (string, error) {
	selected := ""
	source := strings.TrimSpace(metadata.Name)
	if metadata.Margo.Site != nil {
		selected = strings.TrimSpace(metadata.Margo.Site.Layout)
	}
	if selected == "" {
		selected = strings.TrimSpace(family.Layout)
	}
	if selected == "" {
		selected = strings.TrimSpace(layouts.Default)
	}
	if selected == "" {
		return "", presentationSourceDiagnostic(newPresentationDiagnostic(
			"site.layout_required",
			"no layout profile was selected",
			"Set layouts.default, a family layout, or margo.site.layout.",
			"/layouts/default",
		), source)
	}
	if _, ok := layouts.Profiles[selected]; !ok {
		profileSource := "layouts.default"
		pointer := "/layouts/default"
		if metadata.Margo.Site != nil && strings.TrimSpace(metadata.Margo.Site.Layout) != "" {
			profileSource = "margo.site.layout"
			pointer = "/margo/site/layout"
		} else if strings.TrimSpace(family.Layout) != "" {
			profileSource = "family layout"
			pointer = "/navigation/families/layout"
		}
		hint := fmt.Sprintf("Select configured profile %q for %s", selected, profileSource)
		if source != "" {
			hint += fmt.Sprintf(" on source %q", source)
		}
		if family.ID != "" {
			hint += fmt.Sprintf(" in family %q", family.ID)
		}
		return "", presentationSourceDiagnostic(newPresentationDiagnostic(
			"site.layout_unknown",
			fmt.Sprintf("layout profile %q is not configured", selected),
			hint+".",
			pointer,
		), source)
	}
	return selected, nil
}

func sourceLocaleForPresentation(source string, locales LocaleConfig) (string, string) {
	if len(locales.Supported) > 0 {
		return sourceLocale(source, locales)
	}
	if locales.Default != "" {
		return sourceLocale(source, LocaleConfig{Default: locales.Default, Supported: []string{locales.Default}})
	}
	return "", source
}

func normalizeFamilyRootForMatch(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	if source == "." {
		return "."
	}
	return path.Clean(source)
}

func familyRootMatches(root, source string) bool {
	if root == "." {
		return true
	}
	return source == root || strings.HasPrefix(source, root+"/")
}

func familyRootDepth(root string) int {
	if root == "." || root == "" {
		return 0
	}
	return len(strings.Split(root, "/"))
}

func newPresentationDiagnostic(code, message, hint, pointer string) error {
	err := diagnostic(code, message, hint, "")
	var diagnosticError *margo.DiagnosticError
	if errors.As(err, &diagnosticError) && pointer != "" {
		for index := range diagnosticError.Diagnostics {
			diagnosticError.Diagnostics[index].Pointer = pointer
		}
	}
	return err
}

func presentationSourceDiagnostic(err error, source string) error {
	if strings.TrimSpace(source) == "" {
		return err
	}
	var diagnosticError *margo.DiagnosticError
	if errors.As(err, &diagnosticError) {
		for index := range diagnosticError.Diagnostics {
			diagnosticError.Diagnostics[index].Source = source
		}
	}
	return err
}

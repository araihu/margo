package margo

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"
)

// HTMLPageInput configures a generic complete HTML document. Head, Header,
// BeforeContent, and Footer are caller-owned composition seams; Margo does not
// infer canonical URLs, social metadata, or publication semantics here.
type HTMLPageInput struct {
	Theme           ThemeName
	ColorMode       ColorMode
	DependencyMode  HTMLDependencyMode
	ThemeStylesheet AssetRef
	Head            templ.Component
	Header          templ.Component
	BeforeContent   templ.Component
	Footer          templ.Component
	body            templ.Component
	legacyStyles    map[string]string
}

func RenderHTMLPage(result *HTMLResult, input HTMLPageInput) (templ.Component, error) {
	validated, err := validateHTMLPageInput(result, input)
	if err != nil {
		return nil, err
	}
	return htmlPageDocument(
		validated.metadata,
		validated.input,
		validated.dependencies,
		result.Fingerprint().String(),
		result.Fragment(),
	), nil
}

type validatedHTMLPage struct {
	metadata     HTMLMetadata
	input        HTMLPageInput
	dependencies []htmlDependency
}

func validateHTMLPageInput(result *HTMLResult, input HTMLPageInput) (validatedHTMLPage, error) {
	if result == nil || result.Fragment() == nil {
		return validatedHTMLPage{}, htmlError("html.result_required", "HTML result is required")
	}
	metadata := result.Metadata()
	if strings.TrimSpace(metadata.Title) == "" {
		return validatedHTMLPage{}, htmlError("html.metadata_invalid", "page title is required")
	}
	if err := validateThemeName(input.Theme); err != nil {
		return validatedHTMLPage{}, err
	}
	if err := validateColorMode(input.ColorMode); err != nil {
		return validatedHTMLPage{}, err
	}
	if input.DependencyMode != HTMLDependenciesLocal && input.DependencyMode != HTMLDependenciesInline {
		return validatedHTMLPage{}, fmt.Errorf("html.dependency_mode_invalid")
	}
	dependencies, err := resolveHTMLPageDependencies(result.Requirements(), input)
	if err != nil {
		return validatedHTMLPage{}, err
	}
	return validatedHTMLPage{
		metadata:     metadata.clone(),
		input:        input,
		dependencies: append([]htmlDependency(nil), dependencies...),
	}, nil
}

func resolveHTMLPageDependencies(requirements HTMLRequirements, input HTMLPageInput) ([]htmlDependency, error) {
	resolved := make([]htmlDependency, 0, len(requirements.List())+1)
	for _, requirement := range requirements.List() {
		dependency, err := resolveHTMLDependency(requirement, input.DependencyMode, input.legacyStyles[requirement.ID])
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, dependency)
	}

	if assetRefIsZero(input.ThemeStylesheet) {
		if !isBuiltInTheme(input.Theme) {
			return nil, htmlError("html.theme_invalid", fmt.Sprintf("custom theme %q requires a stylesheet", input.Theme))
		}
		return resolved, nil
	}
	if err := input.ThemeStylesheet.validate(); err != nil {
		return nil, htmlError("html.theme_invalid", err.Error())
	}
	if input.ThemeStylesheet.MediaType != "text/css" {
		return nil, htmlError("html.theme_invalid", "theme stylesheet must be CSS")
	}
	themeRequirement := HTMLRequirement{
		ID: "margo.theme." + string(input.Theme), Kind: HTMLStylesheet,
		LocalURL: "/" + input.ThemeStylesheet.Path, Inline: input.ThemeStylesheet.clone(),
	}
	dependency, err := resolveHTMLDependency(themeRequirement, input.DependencyMode, input.legacyStyles[themeRequirement.ID])
	if err != nil {
		return nil, htmlError("html.theme_invalid", err.Error())
	}
	return append(resolved, dependency), nil
}

func htmlPageLanguage(metadata HTMLMetadata) string {
	if metadata.Language == "" {
		return "en"
	}
	return metadata.Language
}

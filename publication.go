package margo

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/a-h/templ"
)

type PublicationKind string

const (
	PublicationDocument PublicationKind = "document"
	PublicationArticle  PublicationKind = "article"
)

type HTMLDependencyMode string

const (
	HTMLDependenciesLocal  HTMLDependencyMode = "local"
	HTMLDependenciesInline HTMLDependencyMode = "inline"
)

type PublicationInput struct {
	Mode            PublicationMode
	Kind            PublicationKind
	Authority       AuthorityRecord
	RoutePath       string
	SiteName        string
	Locale          string
	Image           SocialImage
	Theme           ThemeName
	ColorMode       ColorMode
	DependencyMode  HTMLDependencyMode
	ThemeStylesheet AssetRef
	Header          templ.Component
	Footer          templ.Component
	body            templ.Component
	legacyStyles    map[string]string
}

type publicationDependency struct {
	ID               string
	Kind             HTMLRequirementKind
	URL              string
	Integrity        string
	Inline           string
	LegacyStylesheet string
}

var (
	inlineScriptClose = regexp.MustCompile(`(?i)</script`)
	inlineStyleClose  = regexp.MustCompile(`(?i)</style`)
)

func RenderPublication(editorial *EditorialResult, input PublicationInput) (templ.Component, error) {
	validated, err := validatePublicationInput(editorial, input)
	if err != nil {
		return nil, err
	}
	return publicationDocument(
		validated.metadata,
		validated.input,
		validated.social,
		validated.dependencies,
		editorial.Fingerprint().String(),
		editorial.Fragment(),
	), nil
}

type validatedPublication struct {
	metadata     EditorialMetadata
	input        PublicationInput
	social       SocialMetadata
	dependencies []publicationDependency
}

func validatePublicationInput(editorial *EditorialResult, input PublicationInput) (validatedPublication, error) {
	if editorial == nil || editorial.Fragment() == nil {
		return validatedPublication{}, editorialError("editorial.result_required", "editorial result is required")
	}
	metadata := editorial.Metadata()
	if strings.TrimSpace(metadata.Title) == "" {
		return validatedPublication{}, editorialError("editorial.metadata_invalid", "publication title is required")
	}
	if input.Kind != PublicationDocument && input.Kind != PublicationArticle {
		return validatedPublication{}, fmt.Errorf("publication.kind_invalid")
	}
	if input.Mode != PublicationPrivate && input.Mode != PublicationPublic {
		return validatedPublication{}, fmt.Errorf("publication.mode_invalid")
	}
	if err := validateThemeName(input.Theme); err != nil {
		return validatedPublication{}, err
	}
	if err := validateColorMode(input.ColorMode); err != nil {
		return validatedPublication{}, err
	}
	if input.DependencyMode != HTMLDependenciesLocal && input.DependencyMode != HTMLDependenciesInline {
		return validatedPublication{}, fmt.Errorf("publication.dependency_mode_invalid")
	}

	dependencies, err := resolvePublicationDependencies(editorial.Requirements(), input)
	if err != nil {
		return validatedPublication{}, err
	}

	social := SocialMetadata{
		Title: metadata.Title, Description: metadata.Description,
		SiteName: input.SiteName, Locale: input.Locale, Image: input.Image,
	}
	if input.Mode == PublicationPublic {
		social.CanonicalURL = strings.TrimRight(string(input.Authority.CanonicalOrigin), "/") + input.RoutePath
	}
	if err := social.Validate(input.Mode, input.Authority, input.RoutePath); err != nil {
		return validatedPublication{}, err
	}

	return validatedPublication{
		metadata: metadata.clone(), input: input, social: social,
		dependencies: append([]publicationDependency(nil), dependencies...),
	}, nil
}

func resolvePublicationDependencies(requirements HTMLRequirements, input PublicationInput) ([]publicationDependency, error) {
	resolved := make([]publicationDependency, 0, len(requirements.List())+1)
	for _, requirement := range requirements.List() {
		dependency, err := resolvePublicationDependency(requirement, input.DependencyMode, input.legacyStyles[requirement.ID])
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, dependency)
	}

	if assetRefIsZero(input.ThemeStylesheet) {
		if !isBuiltInTheme(input.Theme) {
			return nil, editorialError("editorial.theme_invalid", fmt.Sprintf("custom theme %q requires a stylesheet", input.Theme))
		}
		return resolved, nil
	}
	if err := input.ThemeStylesheet.validate(); err != nil {
		return nil, editorialError("editorial.theme_invalid", err.Error())
	}
	if input.ThemeStylesheet.MediaType != "text/css" {
		return nil, editorialError("editorial.theme_invalid", "theme stylesheet must be CSS")
	}
	themeRequirement := HTMLRequirement{
		ID: "margo.theme." + string(input.Theme), Kind: HTMLStylesheet,
		LocalURL: "/" + input.ThemeStylesheet.Path, Inline: input.ThemeStylesheet.clone(),
	}
	dependency, err := resolvePublicationDependency(themeRequirement, input.DependencyMode, input.legacyStyles[themeRequirement.ID])
	if err != nil {
		return nil, editorialError("editorial.theme_invalid", err.Error())
	}
	return append(resolved, dependency), nil
}

func resolvePublicationDependency(requirement HTMLRequirement, mode HTMLDependencyMode, legacyStylesheet string) (publicationDependency, error) {
	dependency := publicationDependency{ID: requirement.ID, Kind: requirement.Kind, Integrity: requirement.Integrity, LegacyStylesheet: legacyStylesheet}
	switch mode {
	case HTMLDependenciesLocal:
		if requirement.LocalURL == "" {
			return publicationDependency{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has no local URL", requirement.ID))
		}
		dependency.URL = requirement.LocalURL
	case HTMLDependenciesInline:
		if assetRefIsZero(requirement.Inline) || len(requirement.Inline.Content) == 0 {
			return publicationDependency{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has no inline bytes", requirement.ID))
		}
		if requirement.Kind == HTMLStylesheet {
			dependency.Inline = inlineStyleClose.ReplaceAllString(string(requirement.Inline.Content), `<\/style`)
		} else {
			dependency.Inline = inlineScriptClose.ReplaceAllString(string(requirement.Inline.Content), `<\/script`)
		}
	}
	return dependency, nil
}

func publicationOGType(kind PublicationKind) string {
	if kind == PublicationArticle {
		return "article"
	}
	return "website"
}

func publicationLanguage(metadata EditorialMetadata) string {
	if metadata.Language == "" {
		return "en"
	}
	return metadata.Language
}

func publicationDependencyComponent(dependency publicationDependency) templ.Component {
	attribute := func(name, value string) string {
		if value == "" {
			return ""
		}
		return " " + name + `="` + html.EscapeString(value) + `"`
	}
	common := attribute("data-margo-stylesheet", dependency.LegacyStylesheet) + attribute("data-margo-requirement", dependency.ID)
	if dependency.Kind == HTMLStylesheet {
		if dependency.Inline != "" {
			return templ.Raw("<style" + common + ">" + dependency.Inline + "</style>")
		}
		return templ.Raw(`<link rel="stylesheet"` + attribute("href", dependency.URL) + attribute("integrity", dependency.Integrity) + func() string {
			if dependency.Integrity == "" {
				return ""
			}
			return ` crossorigin="anonymous"`
		}() + common + ">")
	}
	if dependency.Inline != "" {
		return templ.Raw("<script" + common + ">" + dependency.Inline + "</script>")
	}
	return templ.Raw("<script defer" + attribute("src", dependency.URL) + attribute("integrity", dependency.Integrity) + func() string {
		if dependency.Integrity == "" {
			return ""
		}
		return ` crossorigin="anonymous"`
	}() + common + "></script>")
}

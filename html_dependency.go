package margo

import (
	"fmt"
	"html"
	"regexp"

	"github.com/a-h/templ"
)

type HTMLDependencyMode string

const (
	HTMLDependenciesLocal  HTMLDependencyMode = "local"
	HTMLDependenciesInline HTMLDependencyMode = "inline"
)

type htmlDependency struct {
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

func resolveHTMLDependency(requirement HTMLRequirement, mode HTMLDependencyMode, legacyStylesheet string) (htmlDependency, error) {
	dependency := htmlDependency{ID: requirement.ID, Kind: requirement.Kind, Integrity: requirement.Integrity, LegacyStylesheet: legacyStylesheet}
	switch mode {
	case HTMLDependenciesLocal:
		if requirement.LocalURL == "" {
			return htmlDependency{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has no local URL", requirement.ID))
		}
		dependency.URL = requirement.LocalURL
	case HTMLDependenciesInline:
		if assetRefIsZero(requirement.Inline) || len(requirement.Inline.Content) == 0 {
			return htmlDependency{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has no inline bytes", requirement.ID))
		}
		if requirement.Kind == HTMLStylesheet {
			dependency.Inline = inlineStyleClose.ReplaceAllString(string(requirement.Inline.Content), `<\/style`)
		} else {
			dependency.Inline = inlineScriptClose.ReplaceAllString(string(requirement.Inline.Content), `<\/script`)
		}
	}
	return dependency, nil
}

func htmlDependencyComponent(dependency htmlDependency) templ.Component {
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

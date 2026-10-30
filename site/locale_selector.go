package site

import (
	"html"

	"github.com/a-h/templ"
)

func (config LocaleConfig) selectorEnabled() bool {
	return config.Selector.Enabled == nil || *config.Selector.Enabled
}

func localeName(locale string) string {
	switch locale {
	case "en":
		return "English"
	case "pt-BR":
		return "Português"
	case "es":
		return "Español"
	default:
		return locale
	}
}

func (b *builder) localeFlagHTML(locale string) string {
	flag := b.config.Locales.Selector.Flags[locale]
	if flag == "" || !b.config.Locales.selectorEnabled() {
		return ""
	}
	return `<img class="margo-locale-flag" src="` + html.EscapeString(b.publicationArtifactHref(flag)) + `" width="24" height="18" alt="" aria-hidden="true" style="object-fit:contain">`
}

func (b *builder) localeFlag(locale string) templ.Component {
	if markup := b.localeFlagHTML(locale); markup != "" {
		return templ.Raw(markup)
	}
	return nil
}

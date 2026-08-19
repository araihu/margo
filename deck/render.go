package deck

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/araihu/margo"
)

type RenderInput struct {
	Name      string
	Markdown  []byte
	BaseURL   string
	Theme     margo.ThemeName
	ColorMode margo.ColorMode
	Geometry  DeckGeometry
}

type Result struct {
	html              []byte
	slideCount        int
	fingerprint       margo.DocumentFingerprint
	slideResults      []*margo.RenderResult
	theme             margo.ThemeName
	colorMode         margo.ColorMode
	geometry          DeckGeometry
	lang              string
	metadata          Metadata
	requirements      margo.HTMLRequirements
	validationRequest margo.RuntimeValidationRequest
}

func Render(ctx context.Context, compiler *margo.Compiler, input RenderInput, options ...RenderOption) (*Result, error) {
	if compiler == nil {
		return nil, deckError("deck.compiler_required", input.Name, 1, "compiler is required")
	}
	if !compiler.SupportsRenderIDAllocator() {
		return nil, deckError("deck.extension_id_unsafe", input.Name, 1, "registered extensions must declare NamespacedIDsV1 for deck rendering")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	renderOptions, err := applyRenderOptions(options)
	if err != nil {
		return nil, err
	}
	document, err := Parse(input.Name, input.Markdown)
	if err != nil {
		return nil, err
	}
	theme, colorMode, err := resolvePresentation(input, renderOptions, document.Directives())
	if err != nil {
		return nil, err
	}
	geometry, err := resolveGeometry(input, renderOptions, document.Directives())
	if err != nil {
		return nil, err
	}
	validationRequest, err := resolveValidationRequest(theme, renderOptions.validationRequest)
	if err != nil {
		return nil, err
	}
	slides := document.Slides()
	allocator := renderOptions.idAllocator
	if allocator == nil {
		allocator = NewRenderIDAllocator("margo-deck")
	}
	fragments := make([][]byte, len(slides))
	slotFragments := make([][][]byte, len(slides))
	runtimeResults := make([]*margo.RenderResult, 0, len(slides))
	requirementGroups := make([]margo.HTMLRequirements, len(slides))
	for index, slide := range slides {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		layout := slide.Layout()
		if layout == nil {
			scope := scopedIDAllocator{root: allocator, scope: fmt.Sprintf("slide-%04d", slide.Ordinal())}
			fragment, renderResult, requirements, err := renderDeckFragment(ctx, compiler, fmt.Sprintf("%s#slide-%d", input.Name, slide.Ordinal()), slide.Markdown(), input.BaseURL, scope)
			if err != nil {
				return nil, err
			}
			fragments[index] = fragment
			runtimeResults = append(runtimeResults, renderResult)
			requirementGroups[index] = requirements
			continue
		}
		slotFragments[index] = make([][]byte, len(layout.Slots))
		results := make([]*margo.RenderResult, 0, len(layout.Slots))
		groups := make([]margo.HTMLRequirements, 0, len(layout.Slots))
		for slotIndex, slot := range layout.Slots {
			scope := scopedIDAllocator{root: allocator, scope: fmt.Sprintf("slide-%04d-slot-%s", slide.Ordinal(), slot.Name)}
			fragment, renderResult, requirements, err := renderDeckFragment(ctx, compiler, fmt.Sprintf("%s#slide-%d-slot-%s", input.Name, slide.Ordinal(), slot.Name), slot.Markdown, input.BaseURL, scope)
			if err != nil {
				return nil, err
			}
			slotFragments[index][slotIndex] = fragment
			results = append(results, renderResult)
			groups = append(groups, requirements)
		}
		runtimeResults = append(runtimeResults, results...)
		requirements, err := margo.MergeHTMLRequirements(groups...)
		if err != nil {
			return nil, err
		}
		requirementGroups[index] = requirements
	}
	requirements, err := margo.MergeHTMLRequirements(requirementGroups...)
	if err != nil {
		return nil, err
	}
	lang := document.Directives().Lang
	article := renderDeckArticle(slides, fragments, slotFragments, geometry, lang)
	page, err := renderDeckPage(document.Metadata(), theme, colorMode, lang, geometry, article, requirements)
	if err != nil {
		return nil, err
	}
	fingerprint := deckFingerprint(input, theme, colorMode, geometry, lang)
	return &Result{
		html:              page,
		slideCount:        len(slides),
		fingerprint:       fingerprint,
		slideResults:      append([]*margo.RenderResult(nil), runtimeResults...),
		theme:             theme,
		colorMode:         colorMode,
		geometry:          geometry,
		lang:              lang,
		metadata:          document.Metadata(),
		requirements:      requirements,
		validationRequest: validationRequest,
	}, nil
}

func renderDeckFragment(ctx context.Context, compiler *margo.Compiler, name string, markdown []byte, baseURL string, allocator margo.RenderIDAllocator) ([]byte, *margo.RenderResult, margo.HTMLRequirements, error) {
	compiled, err := compiler.Compile(ctx, margo.Source{Name: name, Content: markdown, BaseURL: baseURL})
	if err != nil {
		return nil, nil, margo.HTMLRequirements{}, err
	}
	options := []margo.RenderOption{margo.WithRenderTarget(margo.TargetDeck)}
	if allocator != nil {
		options = append(options, margo.WithRenderIDAllocator(allocator))
	}
	renderResult, err := compiler.Render(ctx, compiled, options...)
	if err != nil {
		return nil, nil, margo.HTMLRequirements{}, err
	}
	renderResult, err = margo.PrepareHTMLRenderResult(renderResult)
	if err != nil {
		return nil, nil, margo.HTMLRequirements{}, err
	}
	htmlResult, err := margo.RenderHTML(renderResult)
	if err != nil {
		return nil, nil, margo.HTMLRequirements{}, err
	}
	var fragment bytes.Buffer
	if err := htmlResult.Fragment().Render(ctx, &fragment); err != nil {
		return nil, nil, margo.HTMLRequirements{}, fmt.Errorf("deck.fragment_render: %w", err)
	}
	return append([]byte(nil), fragment.Bytes()...), renderResult, htmlResult.Requirements(), nil
}

func resolvePresentation(input RenderInput, options renderOptions, directives DirectiveState) (margo.ThemeName, margo.ColorMode, error) {
	theme := directives.Theme
	if theme == "" {
		theme = margo.ThemeModern
	}
	if input.Theme != "" {
		if options.theme != nil && *options.theme != input.Theme {
			return "", "", deckError("deck.presentation_conflict", input.Name, 1, "theme API sources disagree")
		}
		theme = input.Theme
	}
	if options.theme != nil {
		theme = *options.theme
	}
	colorMode := directives.ColorMode
	if colorMode == "" {
		colorMode = margo.ColorModeLight
	}
	if input.ColorMode != "" {
		if options.colorMode != nil && *options.colorMode != input.ColorMode {
			return "", "", deckError("deck.presentation_conflict", input.Name, 1, "color mode API sources disagree")
		}
		colorMode = input.ColorMode
	}
	if options.colorMode != nil {
		colorMode = *options.colorMode
	}
	return normalizePresentation(theme, colorMode, input.Name)
}

func resolveGeometry(input RenderInput, options renderOptions, directives DirectiveState) (DeckGeometry, error) {
	fromDocument, err := ParseDeckGeometry(directives.Size)
	if err != nil {
		return DeckGeometry{}, deckError("deck.size_invalid", input.Name, 1, err.Error())
	}
	geometry := fromDocument
	if input.Geometry.Preset != "" {
		if err := input.Geometry.Validate(); err != nil {
			return DeckGeometry{}, err
		}
		if options.geometry != nil && !options.geometry.Equal(input.Geometry) {
			return DeckGeometry{}, deckError("deck.geometry_conflict", input.Name, 1, "RenderInput and WithGeometry overrides disagree")
		}
		geometry = input.Geometry
	}
	if options.geometry != nil {
		geometry = *options.geometry
	}
	return geometry, nil
}

func (r *Result) HTML() []byte {
	if r == nil {
		return nil
	}
	return append([]byte(nil), r.html...)
}

func (r *Result) SlideCount() int {
	if r == nil {
		return 0
	}
	return r.slideCount
}

func (r *Result) DocumentFingerprint() margo.DocumentFingerprint {
	if r == nil {
		return margo.DocumentFingerprint{}
	}
	return r.fingerprint
}

func (r *Result) Requirements() margo.HTMLRequirements {
	if r == nil {
		return margo.HTMLRequirements{}
	}
	return r.requirements
}

// ValidationRequest returns the immutable profile request resolved during
// rendering.
func (r *Result) ValidationRequest() margo.RuntimeValidationRequest {
	if r == nil {
		return margo.RuntimeValidationRequest{}
	}
	return r.validationRequest
}

// Geometry returns the fixed logical canvas selected for the deck.
func (r *Result) Geometry() DeckGeometry {
	if r == nil {
		return DeckGeometry{}
	}
	return r.geometry
}

func renderDeckArticle(slides []Slide, fragments [][]byte, slotFragments [][][]byte, geometry DeckGeometry, documentLang string) []byte {
	var output bytes.Buffer
	_, _ = fmt.Fprintf(&output, `<article class="margo-deck" data-margo-width="%g" data-margo-height="%g" data-margo-unit="%s" data-margo-preset="%s">`, geometry.Width, geometry.Height, html.EscapeString(string(geometry.Unit)), html.EscapeString(geometry.Preset))
	for index, slide := range slides {
		state := " hidden"
		current := ""
		if index == 0 {
			state = ""
			current = ` aria-current="page"`
		}
		classes := []string{"margo-deck__slide"}
		for _, class := range slide.Directives().Classes {
			if validLayoutClass(class) {
				classes = append(classes, "margo-deck__slide--"+class)
			}
		}
		stateDirectives := slide.Directives()
		chapterNumber := 0
		for priorIndex := 0; priorIndex <= index; priorIndex++ {
			for _, class := range slides[priorIndex].Directives().Classes {
				if class == "chapter" {
					chapterNumber++
					break
				}
			}
		}
		sectionLang := strings.TrimSpace(stateDirectives.Lang)
		if sectionLang == "" {
			sectionLang = strings.TrimSpace(documentLang)
		}
		if sectionLang == "" {
			sectionLang = "en"
		}
		ariaLabel := localizedDeckSlideLabel(sectionLang, slide.Ordinal(), len(slides))
		chapterAttribute := ""
		if chapterNumber > 0 {
			for _, class := range stateDirectives.Classes {
				if class == "chapter" {
					ariaLabel = localizedDeckChapterLabel(sectionLang, chapterNumber)
					chapterAttribute = fmt.Sprintf(` data-margo-chapter-label="%s" data-margo-chapter-number="%d"`, html.EscapeString(strings.TrimSuffix(ariaLabel, fmt.Sprintf(" %d", chapterNumber))), chapterNumber)
					break
				}
			}
		}
		langAttribute := ` lang="` + html.EscapeString(sectionLang) + `"`
		colorAttribute := ""
		if stateDirectives.Color != "" && stateDirectives.Color != "transparent" {
			colorAttribute = ` data-margo-color="` + html.EscapeString(stateDirectives.Color) + `"`
		}
		paginateAttribute := ""
		if stateDirectives.Paginate != "" {
			paginateAttribute = ` data-margo-paginate="` + html.EscapeString(stateDirectives.Paginate) + `"`
		}
		_, _ = fmt.Fprintf(&output, `<section id="%s" class="%s" role="region" aria-label="%s" tabindex="-1" data-margo-slide="%d"%s%s%s%s%s%s>`,
			html.EscapeString(slide.ID()), html.EscapeString(strings.Join(classes, " ")), html.EscapeString(ariaLabel), index, current, state, langAttribute, chapterAttribute, colorAttribute, paginateAttribute)
		if stateDirectives.BackgroundColor != "" && stateDirectives.BackgroundColor != "transparent" {
			_, _ = fmt.Fprintf(&output, `<div class="margo-deck__color-layer" data-margo-background-color="%s" aria-hidden="true"></div>`, html.EscapeString(stateDirectives.BackgroundColor))
		}
		if background := stateDirectives.Background; background.Source != "" {
			backgroundAttributes := fmt.Sprintf(` data-margo-background="%s"`, html.EscapeString(background.Source))
			if background.Position != "" {
				backgroundAttributes += ` data-margo-background-position="` + html.EscapeString(background.Position) + `"`
			}
			if background.Repeat != "" {
				backgroundAttributes += ` data-margo-background-repeat="` + html.EscapeString(background.Repeat) + `"`
			}
			if background.Size != "" {
				backgroundAttributes += ` data-margo-background-size="` + html.EscapeString(background.Size) + `"`
			}
			backgroundContent := ""
			if _, gradient := gradientTokens[background.Source]; !gradient {
				backgroundContent = `<img class="margo-deck__background-image" src="` + html.EscapeString(background.Source) + `" alt="" aria-hidden="true">`
			}
			if background.Decorative {
				_, _ = fmt.Fprintf(&output, `<div class="margo-deck__background" aria-hidden="true"%s>%s</div>`, backgroundAttributes, backgroundContent)
			} else {
				_, _ = fmt.Fprintf(&output, `<div class="margo-deck__background" role="img" aria-label="%s"%s>%s</div>`, html.EscapeString(background.Alt), backgroundAttributes, backgroundContent)
			}
		}
		if stateDirectives.Header != "" {
			_, _ = fmt.Fprintf(&output, `<header class="margo-deck__header">%s</header>`, html.EscapeString(stateDirectives.Header))
		}
		layout := slide.Layout()
		if layout == nil {
			_, _ = output.Write(fragments[index])
		} else {
			_, _ = fmt.Fprintf(&output, `<div class="margo-layout margo-layout--%s" data-margo-slot-count="%d"`, html.EscapeString(layout.Class), len(layout.Slots))
			if layout.Class == "metrics" {
				_, _ = fmt.Fprintf(&output, ` role="group" aria-label="%s"`, html.EscapeString(localizedDeckLayoutLabel(sectionLang, layout.Class)))
			}
			_ = output.WriteByte('>')
			if layout.Class == "timeline" {
				_, _ = output.WriteString(`<ol class="margo-layout__slots">`)
			} else {
				_, _ = output.WriteString(`<div class="margo-layout__slots">`)
			}
			for slotIndex, slot := range layout.Slots {
				tag := "div"
				closeTag := "</div>"
				if layout.Class == "timeline" {
					tag, closeTag = "li", "</li>"
				}
				_, _ = fmt.Fprintf(&output, `<%s class="margo-layout__slot margo-layout__slot--%s">`, tag, html.EscapeString(slot.Name))
				if index < len(slotFragments) && slotIndex < len(slotFragments[index]) {
					_, _ = output.Write(slotFragments[index][slotIndex])
				}
				_, _ = output.WriteString(closeTag)
			}
			if layout.Class == "timeline" {
				_, _ = output.WriteString(`</ol>`)
			} else {
				_, _ = output.WriteString(`</div>`)
			}
			_, _ = output.WriteString(`</div>`)
		}
		if stateDirectives.Footer != "" {
			_, _ = fmt.Fprintf(&output, `<footer class="margo-deck__footer">%s</footer>`, html.EscapeString(stateDirectives.Footer))
		}
		if stateDirectives.Paginate != "" && stateDirectives.Paginate != "false" {
			_, _ = fmt.Fprintf(&output, `<span class="margo-deck__pagination" aria-hidden="true">%s</span>`, html.EscapeString(localizedDeckSlideLabel(sectionLang, slide.Ordinal(), len(slides))))
		}
		_, _ = output.WriteString(`</section>`)
	}
	_, _ = output.WriteString(`</article>`)
	return append([]byte(nil), output.Bytes()...)
}

func normalizePresentation(theme margo.ThemeName, colorMode margo.ColorMode, source string) (margo.ThemeName, margo.ColorMode, error) {
	if theme == "" {
		theme = margo.ThemeModern
	}
	if colorMode == "" {
		colorMode = margo.ColorModeLight
	}
	switch theme {
	case margo.ThemeModern, margo.ThemeGoshtoso, margo.ThemeMinimal:
	default:
		return "", "", deckError("deck.theme_invalid", source, 1, "unsupported deck theme")
	}
	switch colorMode {
	case margo.ColorModeLight, margo.ColorModeDark:
	default:
		return "", "", deckError("deck.color_mode_invalid", source, 1, "unsupported deck color mode")
	}
	return theme, colorMode, nil
}

func deckFingerprint(input RenderInput, theme margo.ThemeName, colorMode margo.ColorMode, geometry DeckGeometry, lang string) margo.DocumentFingerprint {
	preimage := []byte(fmt.Sprintf("margo/deck/v2\n%s\n%s\n%s\n%s\n%s\n%g\n%g\n%s\n", input.Name, input.BaseURL, theme, colorMode, lang, geometry.Width, geometry.Height, geometry.Preset))
	preimage = append(preimage, input.Markdown...)
	return margo.DocumentFingerprint(sha256.Sum256(preimage))
}

func slideInstanceID(index int) (margo.RenderInstanceID, error) {
	ordinal := strconv.FormatInt(int64(index), 36)
	if len(ordinal) > 32 {
		return "", deckError("deck.slide_identity_exhausted", "", 1, "slide identity exceeds runtime grammar")
	}
	if len(ordinal) < 8 {
		ordinal = strings.Repeat("0", 8-len(ordinal)) + ordinal
	}
	return margo.RenderInstanceID("ri-" + ordinal), nil
}

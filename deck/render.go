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
}

type Result struct {
	html         []byte
	slideCount   int
	fingerprint  margo.DocumentFingerprint
	slideResults []*margo.RenderResult
	theme        margo.ThemeName
	colorMode    margo.ColorMode
	metadata     Metadata
}

func Render(ctx context.Context, compiler *margo.Compiler, input RenderInput) (*Result, error) {
	if compiler == nil {
		return nil, deckError("deck.compiler_required", input.Name, 1, "compiler is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	theme, colorMode, err := normalizePresentation(input.Theme, input.ColorMode, input.Name)
	if err != nil {
		return nil, err
	}
	document, err := Parse(input.Name, input.Markdown)
	if err != nil {
		return nil, err
	}
	slides := document.Slides()
	fragments := make([][]byte, len(slides))
	rendered := make([]*margo.RenderResult, len(slides))
	for index, slide := range slides {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		compiled, err := compiler.Compile(ctx, margo.Source{
			Name:    fmt.Sprintf("%s#slide-%d", input.Name, slide.Ordinal()),
			Content: slide.Markdown(),
			BaseURL: input.BaseURL,
		})
		if err != nil {
			return nil, err
		}
		renderResult, err := compiler.Render(ctx, compiled)
		if err != nil {
			return nil, err
		}
		htmlResult, err := margo.RenderHTML(renderResult)
		if err != nil {
			return nil, err
		}
		var fragment bytes.Buffer
		if err := htmlResult.Fragment().Render(ctx, &fragment); err != nil {
			return nil, fmt.Errorf("deck.fragment_render: %w", err)
		}
		fragments[index] = append([]byte(nil), fragment.Bytes()...)
		rendered[index] = renderResult
	}
	page := renderDeckArticle(slides, fragments)
	fingerprint := deckFingerprint(input, theme, colorMode)
	return &Result{
		html:         page,
		slideCount:   len(slides),
		fingerprint:  fingerprint,
		slideResults: append([]*margo.RenderResult(nil), rendered...),
		theme:        theme,
		colorMode:    colorMode,
		metadata:     document.Metadata(),
	}, nil
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

func (r *Result) RuntimeDescriptor(instance margo.RenderInstanceID) (margo.RuntimeDescriptor, error) {
	if r == nil {
		return margo.RuntimeDescriptor{}, deckError("deck.result_required", "", 1, "deck result is required")
	}
	parts := make([]margo.RuntimeDescriptor, len(r.slideResults))
	for index, result := range r.slideResults {
		privateInstance, err := slideInstanceID(index)
		if err != nil {
			return margo.RuntimeDescriptor{}, err
		}
		parts[index], err = result.RuntimeDescriptor(privateInstance)
		if err != nil {
			return margo.RuntimeDescriptor{}, err
		}
	}
	return margo.ComposeRuntimeDescriptors(r.fingerprint, instance, parts...)
}

func renderDeckArticle(slides []Slide, fragments [][]byte) []byte {
	var output bytes.Buffer
	_, _ = output.WriteString(`<article class="margo-deck">`)
	for index, slide := range slides {
		_, _ = fmt.Fprintf(&output, `<section id="%s" class="margo-deck__slide" role="region" aria-label="Slide %d of %d" tabindex="-1" data-margo-slide="%d">`,
			html.EscapeString(slide.ID()), slide.Ordinal(), len(slides), index)
		_, _ = output.Write(fragments[index])
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

func deckFingerprint(input RenderInput, theme margo.ThemeName, colorMode margo.ColorMode) margo.DocumentFingerprint {
	preimage := []byte("margo/deck/v1\n" + input.Name + "\n" + input.BaseURL + "\n" + string(theme) + "\n" + string(colorMode) + "\n")
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

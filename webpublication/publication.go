// Package webpublication adds public-web authority, canonical URL, social
// metadata, and article conventions around Margo's generic HTML page contract.
package webpublication

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/margo"
)

type Kind string

const (
	KindDocument Kind = "document"
	KindArticle  Kind = "article"
)

type Input struct {
	Kind      Kind
	Authority AuthorityRecord
	RoutePath string
	SiteName  string
	Locale    string
	Image     SocialImage
	Page      margo.HTMLPageInput
}

func Render(result *margo.HTMLResult, input Input) (templ.Component, error) {
	if result == nil || result.Fragment() == nil {
		return nil, fmt.Errorf("webpublication.result_required")
	}
	if input.Kind != KindDocument && input.Kind != KindArticle {
		return nil, fmt.Errorf("webpublication.kind_invalid")
	}
	metadata := result.Metadata()
	social := SocialMetadata{
		Title: metadata.Title, Description: metadata.Description,
		CanonicalURL: strings.TrimRight(string(input.Authority.CanonicalOrigin), "/") + input.RoutePath,
		SiteName:     input.SiteName, Locale: input.Locale, Image: input.Image,
	}
	if err := social.Validate(input.Authority, input.RoutePath); err != nil {
		return nil, err
	}

	page := input.Page
	page.Head = joinComponents(page.Head, publicationMetadataTags(social, input.Kind, metadata))
	if input.Kind == KindArticle {
		page.BeforeContent = joinComponents(publicationArticleDetails(metadata), page.BeforeContent)
	}
	return margo.RenderHTMLPage(result, page)
}

func joinComponents(components ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		for _, component := range components {
			if component == nil {
				continue
			}
			if err := component.Render(ctx, writer); err != nil {
				return err
			}
		}
		return nil
	})
}

func formatInt(value int) string { return strconv.Itoa(value) }

func openGraphType(kind Kind) string {
	if kind == KindArticle {
		return "article"
	}
	return "website"
}

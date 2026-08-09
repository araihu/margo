package site

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/margo"
)

// blogPublicationInput belongs to this example consumer. It intentionally is
// not part of Margo's API: canonical routes, social cards, and article chrome
// are site policy composed through Margo's generic page seams.
type blogPublicationInput struct {
	CanonicalURL  string
	SiteName      string
	Locale        string
	ImageURL      string
	ImageMIMEType string
	ImageWidth    int
	ImageHeight   int
	ImageAlt      string
	Page          margo.HTMLPageInput
}

func renderBlogPublication(result *margo.HTMLResult, input blogPublicationInput) (templ.Component, error) {
	if result == nil || result.Fragment() == nil {
		return nil, fmt.Errorf("blog publication result is required")
	}
	if err := validateBlogPublicationInput(input); err != nil {
		return nil, err
	}

	metadata := result.Metadata()
	page := input.Page
	page.Head = joinBlogComponents(page.Head, blogPublicationMetadata(input, metadata))
	page.BeforeContent = joinBlogComponents(blogPublicationDetails(metadata), page.BeforeContent)
	return margo.RenderHTMLPage(result, page)
}

func validateBlogPublicationInput(input blogPublicationInput) error {
	for name, value := range map[string]string{
		"canonical URL": input.CanonicalURL,
		"site name":     input.SiteName,
		"image URL":     input.ImageURL,
		"image MIME":    input.ImageMIMEType,
		"image alt":     input.ImageAlt,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("blog publication %s is required", name)
		}
	}
	for name, value := range map[string]string{"canonical URL": input.CanonicalURL, "image URL": input.ImageURL} {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return fmt.Errorf("blog publication %s must be an absolute HTTPS URL", name)
		}
	}
	if input.ImageWidth <= 0 || input.ImageHeight <= 0 {
		return fmt.Errorf("blog publication image dimensions must be positive")
	}
	return nil
}

func joinBlogComponents(components ...templ.Component) templ.Component {
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

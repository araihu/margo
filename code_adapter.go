package margo

import (
	"context"
	"io"
	"strings"

	"github.com/araihu/goshtoso/components/codeblock"
)

func renderCodeBlock(ctx context.Context, out io.Writer, language string, code []byte) error {
	disableCopyButton := strings.HasSuffix(language, ":copy_disabled")
	if disableCopyButton {
		language = strings.TrimSuffix(language, ":copy_disabled")
	}

	component := codeblock.CodeBlock(codeblock.Config{
		Language:          language,
		Code:              string(code),
		DisableCopyButton: disableCopyButton,
	})
	return component.Render(ctx, out)
}

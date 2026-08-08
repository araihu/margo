package margo

import (
	"context"
	"io"

	"github.com/araihu/goshtoso/components/codeblock"
)

func renderCodeBlock(ctx context.Context, out io.Writer, language string, code []byte) error {
	return codeblock.CodeBlock(codeblock.Config{
		Language: language,
		Code:     string(code),
	}).Render(ctx, out)
}

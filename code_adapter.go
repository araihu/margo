package margo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/araihu/goshtoso/components/codeblock"
)

func renderCodeBlock(ctx context.Context, out io.Writer, language string, code []byte) error {
	copyButton := true
	if strings.HasSuffix(language, ":copy_disabled") {
		language = strings.TrimSuffix(language, ":copy_disabled")
		copyButton = false
	}

	component := codeblock.CodeBlock(codeblock.Config{
		Language:          language,
		Code:              string(code),
		DisableCopyButton: !copyButton,
	})
	if copyButton {
		var markup bytes.Buffer
		if err := component.Render(ctx, &markup); err != nil {
			return err
		}
		marked, err := markCodeBlockCopyControls(markup.String())
		if err != nil {
			return err
		}
		_, err = io.WriteString(out, marked)
		return err
	}

	return component.Render(ctx, out)
}

func markCodeBlockCopyControls(markup string) (string, error) {
	const (
		rootMarker   = ` data-code-block data-density=`
		buttonMarker = ` data-code-block-copy `
		labelMarker  = ` data-code-block-copy-status `
	)
	if !strings.Contains(markup, rootMarker) {
		return "", fmt.Errorf("code block copy button: root marker not found")
	}
	if !strings.Contains(markup, buttonMarker) {
		return "", fmt.Errorf("code block copy button: button marker not found")
	}
	if !strings.Contains(markup, labelMarker) {
		return "", fmt.Errorf("code block copy button: label marker not found")
	}
	marked := strings.Replace(markup, rootMarker, ` data-code-block data-margo-code-copy data-density=`, 1)
	// Margo's standalone runtime owns these controls. Remove the upstream
	// runtime selector so a host that loads both runtimes cannot bind twice.
	marked = strings.Replace(marked, buttonMarker, ` data-margo-code-copy-button `, 1)
	marked = strings.Replace(marked, labelMarker, ` data-margo-code-copy-label `, 1)
	return marked, nil
}

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
		Language: language,
		Code:     string(code),
	})
	if copyButton {
		return component.Render(ctx, out)
	}

	var markup bytes.Buffer
	if err := component.Render(ctx, &markup); err != nil {
		return err
	}
	withoutCopy, err := removeCodeBlockCopyButton(markup.String())
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, withoutCopy)
	return err
}

func removeCodeBlockCopyButton(markup string) (string, error) {
	const buttonStart = ` <button type="button" @click="copyCode()"`
	const buttonEnd = `</button>`

	start := strings.Index(markup, buttonStart)
	if start < 0 {
		return "", fmt.Errorf("code block copy button: start marker not found")
	}
	relativeEnd := strings.Index(markup[start:], buttonEnd)
	if relativeEnd < 0 {
		return "", fmt.Errorf("code block copy button: end marker not found")
	}
	end := start + relativeEnd + len(buttonEnd)
	return markup[:start] + markup[end:], nil
}

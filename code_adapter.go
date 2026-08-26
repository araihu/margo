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

func markCodeBlockCopyControls(markup string) (string, error) {
	const (
		rootMarker   = ` data-code-block data-density=`
		buttonMarker = `<button type="button" @click="copyCode()"`
		labelMarker  = `<span x-text="copied ? 'Copied!' : 'Copy'">Copy</span>`
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
	marked = strings.Replace(marked, buttonMarker, `<button type="button" @click="copyCode()" data-margo-code-copy-button`, 1)
	marked = strings.Replace(marked, labelMarker, `<span x-text="copied ? 'Copied!' : 'Copy'" data-margo-code-copy-label aria-live="polite">Copy</span>`, 1)
	return marked, nil
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

package margo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"
	"sync"

	"github.com/araihu/margo/assets"
	"github.com/araihu/margo/internal/mathsyntax"
	"github.com/yuin/goldmark/ast"
)

const mathEngineVersion = mathsyntax.EngineVersion

var mathIdentityCache struct {
	sync.Once
	identity string
	err      error
}

func mathIdentity() (string, error) {
	mathIdentityCache.Do(func() {
		requirements, err := mathHTMLRequirements()
		if err != nil {
			mathIdentityCache.err = err
			return
		}
		h := sha256.New()
		for _, r := range requirements {
			h.Write([]byte(r.ID))
			h.Write([]byte{0})
			h.Write(r.Inline.Content)
		}
		mathIdentityCache.identity = hex.EncodeToString(h.Sum(nil))
	})
	return mathIdentityCache.identity, mathIdentityCache.err
}

// WithMath enables LaTeX math expressions inside Markdown. The default is false.
func WithMath(enabled bool) Option {
	return func(c *compilerConfig) error {
		delete(c.values, "math")
		if !enabled {
			return nil
		}
		identity, err := mathIdentity()
		if err != nil {
			return err
		}
		c.values["math"] = map[string]any{"profile": mathsyntax.Version, "engine": mathEngineVersion, "assets": identity, "maxExpressionBytes": mathsyntax.MaxExpressionBytes, "maxExpressions": mathsyntax.MaxExpressions, "maxPayloadBytes": mathsyntax.MaxPayloadBytes}
		return nil
	}
}

func mathEnabled(c compilerConfig) bool { _, ok := c.values["math"]; return ok }

// MathEnabled reports the compiler's immutable math setting.
func (c *Compiler) MathEnabled() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return mathEnabled(c.config)
}

// WithCheckMath selects the same math grammar used by WithMath without launching a browser.
func WithCheckMath(enabled bool) CheckOption {
	return func(c *checkConfig) error { c.math = enabled; return nil }
}

func validateMathRegistration(c compilerConfig) error {
	if !mathEnabled(c) {
		return nil
	}
	for _, r := range registryFromConfig(c).registrations {
		for _, f := range r.Fences {
			if f == "math" {
				return fmt.Errorf("math.fence_conflict: host extension %s claims math", r.Identity.Name)
			}
		}
	}
	return nil
}

func mathSourceDiagnostics(source Source, offset int, root ast.Node) []Diagnostic {
	var result []Diagnostic
	for _, issue := range mathsyntax.Validate(root) {
		absolute := min(len(source.Content), offset+issue.Offset)
		line := lineAtOffset(source.Content, absolute)
		start := strings.LastIndexByte(string(source.Content[:absolute]), '\n') + 1
		result = append(result, Diagnostic{Code: issue.Code, Severity: SeverityError, Source: source.Name, Line: line, Column: absolute - start + 1, Pointer: "/math", Message: issue.Message, Hint: "Check math delimiters and the equation source limits."})
	}
	return result
}

type mathTask struct {
	InputSHA256 string
	Position    SourcePosition
}

func compileMathTask(source Source, bodyOffset int, data *mathsyntax.Data, ordinal uint32) (mathTask, error) {
	data.Ordinal = ordinal
	identity, err := mathIdentity()
	if err != nil {
		return mathTask{}, err
	}
	input, err := json.Marshal(struct{ Profile, Engine, Assets, Mode, Payload string }{mathsyntax.Version, mathEngineVersion, identity, map[bool]string{true: "display", false: "inline"}[data.Display], string(data.Payload)})
	if err != nil {
		return mathTask{}, err
	}
	h := sha256.Sum256(input)
	data.InputSHA256 = hex.EncodeToString(h[:])
	offset := min(len(source.Content), bodyOffset+data.Offset)
	line := lineAtOffset(source.Content, offset)
	start := strings.LastIndexByte(string(source.Content[:offset]), '\n') + 1
	data.SourceName = source.Name
	data.Line = line
	data.Column = offset - start + 1
	return mathTask{data.InputSHA256, SourcePosition{Source: source.Name, Line: line, Column: offset - start + 1}}, nil
}

func (r markdownRenderer) renderMath(data *mathsyntax.Data) error {
	tag, mode := "span", "inline"
	if data.Display {
		tag, mode = "div", "display"
	}
	id := fmt.Sprintf("margo-math-%d", data.Ordinal)
	if r.idAllocator != nil {
		id = r.idAllocator.Allocate("math", fmt.Sprint(data.Ordinal))
	}
	_, err := fmt.Fprintf(r.out, `<%s class="margo-math margo-math--%s" id="%s" data-margo-runtime-task="math" data-margo-math-mode="%s" data-margo-math-ordinal="%d" data-margo-math-input="%s" data-margo-output-limit="%d" data-margo-source="%s" data-margo-line="%d" data-margo-column="%d"><span class="margo-math__source">%s</span><span class="margo-math__output"></span><span class="margo-math__error" hidden></span></%s>`, tag, mode, html.EscapeString(id), mode, data.Ordinal, data.InputSHA256, r.policy.OutputBytes, html.EscapeString(data.SourceName), data.Line, data.Column, html.EscapeString(string(data.Payload)), tag)
	return err
}

func mathHTMLRequirements() ([]HTMLRequirement, error) {
	coordinator, err := runtimeCoordinatorRequirement()
	if err != nil {
		return nil, err
	}
	style, err := EmbeddedAsset("math.css")
	if err != nil {
		return nil, err
	}
	runner, err := EmbeddedAsset("math-run.js")
	if err != nil {
		return nil, err
	}
	file, err := assets.MuambaOpen("katex", "katex-min-js")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	engine, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(engine)
	return []HTMLRequirement{coordinator,
		{ID: "margo.math.styles", Kind: HTMLStylesheet, LocalURL: "/margo-assets/math.css", Inline: style},
		{ID: "margo.math.engine", Kind: HTMLScript, LocalURL: "/margo-assets/katex/" + mathEngineVersion + "/katex.min.js", Inline: AssetRef{Path: "katex/" + mathEngineVersion + "/katex.min.js", MediaType: "application/javascript", SHA256: hex.EncodeToString(h[:]), Content: engine}},
		{ID: "margo.math.execute", Kind: HTMLScript, LocalURL: "/margo-assets/math-run.js", LoadAfter: []string{"margo.math.engine", "margo.math.styles", coordinator.ID}, Inline: runner},
	}, nil
}

func runtimeCoordinatorRequirement() (HTMLRequirement, error) {
	asset, err := EmbeddedAsset("runtime-coordinator.js")
	if err != nil {
		return HTMLRequirement{}, err
	}
	return HTMLRequirement{ID: "margo.runtime.coordinator", Kind: HTMLScript, LocalURL: "/margo-assets/runtime-coordinator.js", Inline: asset}, nil
}

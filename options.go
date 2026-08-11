package margo

import (
	"fmt"
	"reflect"
)

// Option configures a Compiler before it is frozen by New.
type Option func(*compilerConfig) error

// RenderOption configures one immutable render operation.
type RenderOption func(*renderOptions) error

type compilerConfig struct {
	values map[string]any
}

type renderOptions struct {
	values map[string]any
}

// RenderTarget selects one explicit artifact projection without changing the
// target-neutral compiled document.
type RenderTarget string

const (
	TargetHTML RenderTarget = "html"
	TargetSite RenderTarget = "site"
	TargetPDF  RenderTarget = "pdf"
	TargetDeck RenderTarget = "deck"
)

// WithRenderTarget selects iframe and security projection for this render.
// Omission defaults to HTML for backward-compatible library calls.
func WithRenderTarget(target RenderTarget) RenderOption {
	return func(options *renderOptions) error {
		switch target {
		case TargetHTML, TargetSite, TargetPDF, TargetDeck:
			options.values["target"] = target
			return nil
		default:
			return fmt.Errorf("render.target_invalid: unsupported target %q", target)
		}
	}
}

func renderTarget(options renderOptions) RenderTarget {
	if target, ok := options.values["target"].(RenderTarget); ok {
		return target
	}
	return TargetHTML
}

func newCompilerConfig() compilerConfig {
	return compilerConfig{values: map[string]any{"schemaVersion": "margo/compiler-config/v1"}}
}

func (c compilerConfig) clone() compilerConfig {
	values := make(map[string]any, len(c.values))
	for key, value := range c.values {
		values[key] = cloneOptionValue(value)
	}
	return compilerConfig{values: values}
}

func cloneOptionValue(value any) any {
	if value == nil {
		return nil
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return value
		}
		out := make(map[string]any, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = cloneOptionValue(iter.Value().Interface())
		}
		return out
	case reflect.Slice:
		out := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = cloneOptionValue(v.Index(i).Interface())
		}
		return out
	case reflect.Array:
		out := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = cloneOptionValue(v.Index(i).Interface())
		}
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return value
		}
		return cloneOptionValue(v.Elem().Interface())
	default:
		return value
	}
}

func applyOptions(config *compilerConfig, options []Option) error {
	for i, option := range options {
		if option == nil {
			return fmt.Errorf("margo: nil option at index %d", i)
		}
		if err := option(config); err != nil {
			return err
		}
	}
	return nil
}

func applyRenderOptions(options []RenderOption) (renderOptions, error) {
	config := renderOptions{values: make(map[string]any)}
	for i, option := range options {
		if option == nil {
			return renderOptions{}, fmt.Errorf("margo: nil render option at index %d", i)
		}
		if err := option(&config); err != nil {
			return renderOptions{}, err
		}
	}
	return config, nil
}

// Package pdf defines renderer-neutral contracts for exporting Margo HTML to
// PDF.
//
// Hosts provide an Engine, a Request containing materialized HTML and its
// runtime descriptor, and an explicit PageConfig and RelativeLinkPolicy. An
// engine returns PDF bytes together with runtime and engine provenance:
//
//	result, err := engine.Export(ctx, pdf.Request{
//		HTML:          html,
//		Runtime:       descriptor,
//		ExecutionID:   executionID,
//		Page:          page,
//		RelativeLinks: pdf.RelativeLinksStrip,
//	})
//
// Engines never select or fall back to another engine. Package pdf also does
// not discover or download a browser. Use package pdf/engines for deterministic
// discovery, or construct package pdf/chromium with an explicitly selected
// Chromium-family executable.
//
// The zero relative-link policy strips document-relative links so a renderer's
// temporary origin cannot leak into a distributed PDF. PageConfig supports A4
// and Letter pages, portrait and landscape orientation, non-negative margins
// in millimeters, and an optional image-height limit.
//
// Reported browser versions are runtime evidence, not a compatibility minimum.
// Platform-native backends remain behind the pdf/native capability boundary and
// may be compiled out or unavailable.
package pdf

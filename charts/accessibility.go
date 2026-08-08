package charts

import (
	"context"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/a-h/templ"
)

// AccessibleRow is one canonical data row accompanying a static chart.
type AccessibleRow struct {
	Series   string
	Category string
	Sample   string
	Value    string
}

// AccessibleData is the complete text alternative for a chart.
type AccessibleData struct {
	Title string
	Rows  []AccessibleRow
}

// AccessibleRenderPolicy carries the immutable root output ceiling.
type AccessibleRenderPolicy struct {
	MaxOutputBytes int64
}

var accessiblePolicyObserver = struct {
	sync.RWMutex
	fn func(AccessibleRenderPolicy)
}{}

// goshtoso-charts v0.0.2-0.20260803224432-297df2f562e8 uses shared palette
// values while constructing SVG options. Serialize only that upstream render
// call so independent root sessions remain race-free without sharing output.
var chartRenderMu sync.Mutex

// ValidateAccessibleRenderPolicy rejects limits outside the root policy range.
func ValidateAccessibleRenderPolicy(policy AccessibleRenderPolicy) error {
	if policy.MaxOutputBytes < 1 || policy.MaxOutputBytes > 64<<20 {
		return chartDiagnostic("chart.output_limit_invalid", "accessible output byte limit must be between 1 and 67108864")
	}
	return nil
}

// Caption returns the deterministic summary label used by upstream charts.
func Caption(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "Chart"
	}
	return title
}

// EscapeAccessibleField encodes delimiters and ASCII control bytes reversibly.
func EscapeAccessibleField(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", chartDiagnostic("chart.accessibility_field_invalid", "accessible field is not valid UTF-8")
	}
	var out strings.Builder
	for index := 0; index < len(value); index++ {
		b := value[index]
		switch {
		case b == '%':
			out.WriteString("%25")
		case b == '|':
			out.WriteString("%7C")
		case b <= 0x1f || b == 0x7f:
			const hex = "0123456789ABCDEF"
			out.WriteByte('%')
			out.WriteByte(hex[b>>4])
			out.WriteByte(hex[b&0x0f])
		default:
			out.WriteByte(b)
		}
	}
	return out.String(), nil
}

// UnescapeAccessibleField reverses EscapeAccessibleField.
func UnescapeAccessibleField(value string) (string, error) {
	var out strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '%' {
			out.WriteByte(value[index])
			continue
		}
		if index+2 >= len(value) {
			return "", chartDiagnostic("chart.accessibility_field_invalid", "incomplete percent escape")
		}
		hi, okHi := decodeHex(value[index+1])
		lo, okLo := decodeHex(value[index+2])
		if !okHi || !okLo {
			return "", chartDiagnostic("chart.accessibility_field_invalid", "invalid percent escape")
		}
		out.WriteByte(hi<<4 | lo)
		index += 2
	}
	decoded := out.String()
	if !utf8.ValidString(decoded) {
		return "", chartDiagnostic("chart.accessibility_field_invalid", "decoded field is not valid UTF-8")
	}
	return decoded, nil
}

func decodeHex(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	default:
		return 0, false
	}
}

func validateRows(rows []AccessibleRow) error {
	for index, row := range rows {
		if strings.TrimSpace(row.Category) == "" || strings.TrimSpace(row.Value) == "" {
			return chartDiagnostic("chart.accessibility_row_invalid", fmt.Sprintf("row %d requires category and value", index))
		}
		hasSeries := row.Series != ""
		hasSample := row.Sample != ""
		switch {
		case !hasSeries && hasSample:
			return chartDiagnostic("chart.accessibility_row_invalid", fmt.Sprintf("row %d sample requires series", index))
		case hasSeries && !hasSample, !hasSeries && !hasSample, hasSeries && hasSample:
			// These are the three permitted forms.
		default:
			return chartDiagnostic("chart.accessibility_row_invalid", fmt.Sprintf("row %d has an unsupported field form", index))
		}
	}
	return nil
}

// AccessibleText renders the canonical, ordered data table for a chart.
func AccessibleText(title string, rows []AccessibleRow, policy AccessibleRenderPolicy) (string, error) {
	if err := ValidateAccessibleRenderPolicy(policy); err != nil {
		return "", err
	}
	if strings.TrimSpace(title) == "" {
		return "", chartDiagnostic("chart.accessibility_title_invalid", "accessible title is required")
	}
	if err := validateRows(rows); err != nil {
		return "", err
	}
	var out boundedBuffer
	if err := out.init(policy.MaxOutputBytes); err != nil {
		return "", err
	}
	if err := renderAccessibleText(&out, title, rows); err != nil {
		return "", err
	}
	return string(out.Bytes()), nil
}

// WithAccessibleData appends a complete text alternative only after the chart
// and table both fit the effective output limit.
func WithAccessibleData(chart templ.Component, data AccessibleData, policy AccessibleRenderPolicy) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if chart == nil {
			return chartDiagnostic("chart.component_invalid", "chart component is required")
		}
		if err := ValidateAccessibleRenderPolicy(policy); err != nil {
			return err
		}
		observeAccessiblePolicy(policy)
		var combined boundedBuffer
		if err := combined.init(policy.MaxOutputBytes); err != nil {
			return err
		}
		chartRenderMu.Lock()
		renderErr := chart.Render(ctx, &combined)
		chartRenderMu.Unlock()
		if renderErr != nil {
			return renderErr
		}
		if err := renderAccessibleText(&combined, data.Title, data.Rows); err != nil {
			return err
		}
		_, err := out.Write(combined.Bytes())
		return err
	})
}

func observeAccessiblePolicy(policy AccessibleRenderPolicy) {
	accessiblePolicyObserver.RLock()
	observer := accessiblePolicyObserver.fn
	accessiblePolicyObserver.RUnlock()
	if observer != nil {
		observer(policy)
	}
}

// setAccessiblePolicyObserverForTest records the exact policy handed to a
// family wrapper and returns a restoration function for isolated tests.
func setAccessiblePolicyObserverForTest(observer func(AccessibleRenderPolicy)) func() {
	accessiblePolicyObserver.Lock()
	previous := accessiblePolicyObserver.fn
	accessiblePolicyObserver.fn = observer
	accessiblePolicyObserver.Unlock()
	return func() {
		accessiblePolicyObserver.Lock()
		accessiblePolicyObserver.fn = previous
		accessiblePolicyObserver.Unlock()
	}
}

func renderAccessibleText(out io.Writer, title string, rows []AccessibleRow) error {
	if strings.TrimSpace(title) == "" {
		return chartDiagnostic("chart.accessibility_title_invalid", "accessible title is required")
	}
	if err := validateRows(rows); err != nil {
		return err
	}
	if err := writeString(out, `<div class="margo-chart-data" data-margo-chart-data="v1"><table><caption>`); err != nil {
		return err
	}
	if err := writeHTMLValue(out, title); err != nil {
		return err
	}
	if err := writeString(out, `</caption><tbody>`); err != nil {
		return err
	}
	for index, row := range rows {
		if err := writeString(out, fmt.Sprintf(`<tr data-margo-chart-row="%d"><td data-field="series">`, index)); err != nil {
			return err
		}
		if err := writeAccessibleField(out, row.Series); err != nil {
			return err
		}
		if err := writeString(out, `</td><td data-field="category">`); err != nil {
			return err
		}
		if err := writeAccessibleField(out, row.Category); err != nil {
			return err
		}
		if err := writeString(out, `</td><td data-field="sample">`); err != nil {
			return err
		}
		if err := writeAccessibleField(out, row.Sample); err != nil {
			return err
		}
		if err := writeString(out, `</td><td data-field="value">`); err != nil {
			return err
		}
		if err := writeAccessibleField(out, row.Value); err != nil {
			return err
		}
		if err := writeString(out, `</td><td data-field="canonical">`); err != nil {
			return err
		}
		if row.Series != "" {
			if err := writeAccessibleField(out, row.Series); err != nil {
				return err
			}
			if err := writeString(out, `|`); err != nil {
				return err
			}
		}
		if err := writeAccessibleField(out, row.Category); err != nil {
			return err
		}
		if row.Sample != "" {
			if err := writeString(out, `|`); err != nil {
				return err
			}
			if err := writeAccessibleField(out, row.Sample); err != nil {
				return err
			}
		}
		if err := writeString(out, `|`); err != nil {
			return err
		}
		if err := writeAccessibleField(out, row.Value); err != nil {
			return err
		}
		if err := writeString(out, `</td></tr>`); err != nil {
			return err
		}
	}
	return writeString(out, `</tbody></table></div>`)
}

func writeString(out io.Writer, value string) error {
	_, err := io.WriteString(out, value)
	return err
}

func writeHTMLValue(out io.Writer, value string) error {
	if !utf8.ValidString(value) {
		return chartDiagnostic("chart.accessibility_field_invalid", "accessible field is not valid UTF-8")
	}
	for index := 0; index < len(value); index++ {
		var encoded string
		switch value[index] {
		case '&':
			encoded = "&amp;"
		case '<':
			encoded = "&lt;"
		case '>':
			encoded = "&gt;"
		case '\'':
			encoded = "&#39;"
		case '"':
			encoded = "&#34;"
		default:
			if err := writeString(out, value[index:index+1]); err != nil {
				return err
			}
			continue
		}
		if err := writeString(out, encoded); err != nil {
			return err
		}
	}
	return nil
}

func writeAccessibleField(out io.Writer, value string) error {
	if !utf8.ValidString(value) {
		return chartDiagnostic("chart.accessibility_field_invalid", "accessible field is not valid UTF-8")
	}
	const hex = "0123456789ABCDEF"
	for index := 0; index < len(value); index++ {
		b := value[index]
		if b == '%' || b == '|' || b <= 0x1f || b == 0x7f {
			encoded := [3]byte{'%', hex[b>>4], hex[b&0x0f]}
			if err := writeBytes(out, encoded[:]); err != nil {
				return err
			}
			continue
		}
		if err := writeHTMLByte(out, b); err != nil {
			return err
		}
	}
	return nil
}

func writeHTMLByte(out io.Writer, value byte) error {
	switch value {
	case '&':
		return writeString(out, "&amp;")
	case '<':
		return writeString(out, "&lt;")
	case '>':
		return writeString(out, "&gt;")
	case '\'':
		return writeString(out, "&#39;")
	case '"':
		return writeString(out, "&#34;")
	default:
		return writeBytes(out, []byte{value})
	}
}

func writeBytes(out io.Writer, value []byte) error {
	if len(value) == 0 {
		return nil
	}
	_, err := out.Write(value)
	return err
}

// boundedBuffer keeps the wrapper independent of the caller's writer while
// rejecting the first byte over the effective root ceiling.
type boundedBuffer struct {
	data  []byte
	limit int64
}

func (b *boundedBuffer) init(limit int64) error {
	if limit < 1 || limit > 64<<20 {
		return chartDiagnostic("chart.output_limit_invalid", "accessible output byte limit must be between 1 and 67108864")
	}
	b.limit = limit
	b.data = make([]byte, 0, int(limit)+1)
	return nil
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	if int64(len(b.data)) > b.limit-int64(len(value)) {
		return 0, chartDiagnostic("chart.output_limit", "accessible output exceeds the effective output byte limit")
	}
	b.data = append(b.data, value...)
	return len(value), nil
}

func (b *boundedBuffer) Bytes() []byte { return b.data }

// extractAccessibleRows is intentionally test-only. It extracts the canonical
// cell from the fixed v1 markup without treating ARIA labels as data evidence.
func extractAccessibleRows(markup string) []string {
	const tableMarker = `<div class="margo-chart-data" data-margo-chart-data="v1"><table>`
	const rowStart = `<tr data-margo-chart-row="`
	const canonicalStart = `<td data-field="canonical">`
	table := strings.Index(markup, tableMarker)
	if table < 0 {
		return nil
	}
	var indexed = make(map[int]string)
	for offset := table + len(tableMarker); ; {
		start := strings.Index(markup[offset:], rowStart)
		if start < 0 {
			break
		}
		start += offset
		indexStart := start + len(rowStart)
		indexEnd := strings.IndexByte(markup[indexStart:], '"')
		if indexEnd < 0 {
			return nil
		}
		indexEnd += indexStart
		rowIndex, err := strconv.Atoi(markup[indexStart:indexEnd])
		if err != nil || rowIndex < 0 {
			return nil
		}
		if _, exists := indexed[rowIndex]; exists {
			return nil
		}
		cell := strings.Index(markup[start:], canonicalStart)
		if cell < 0 {
			return nil
		}
		cell += start + len(canonicalStart)
		end := strings.Index(markup[cell:], `</td>`)
		if end < 0 {
			return nil
		}
		indexed[rowIndex] = html.UnescapeString(markup[cell : cell+end])
		offset = cell + end + len(`</td>`)
	}
	if len(indexed) == 0 {
		return nil
	}
	rows := make([]string, len(indexed))
	for index := range rows {
		value, ok := indexed[index]
		if !ok {
			return nil
		}
		rows[index] = value
	}
	return rows
}

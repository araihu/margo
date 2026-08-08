package charts

import (
	"bytes"
	"context"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestAccessibleTextIsBoundedAndContainsEveryRow(t *testing.T) {
	rows := []AccessibleRow{
		{Series: "Revenue", Category: "Development", Value: "12"},
		{Series: "Revenue", Category: "Production", Value: "18"},
	}
	out, err := AccessibleText("Revenue", rows, AccessibleRenderPolicy{MaxOutputBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if got := extractAccessibleRows(out); len(got) != 2 || got[0] != "Revenue|Development|12" || got[1] != "Revenue|Production|18" {
		t.Fatalf("accessible rows = %#v", got)
	}
	if strings.Contains(out, "data-chart-runtime") || len(out) > 4096 {
		t.Fatalf("unexpected runtime marker or output size: %d", len(out))
	}
}

func TestAccessiblePolicyBoundariesAndOneByteOverflow(t *testing.T) {
	for _, limit := range []int64{0, -1, math.MaxInt64, (64 << 20) + 1} {
		if err := ValidateAccessibleRenderPolicy(AccessibleRenderPolicy{MaxOutputBytes: limit}); err == nil || !strings.Contains(err.Error(), "chart.output_limit_invalid") {
			t.Fatalf("limit %d accepted: %v", limit, err)
		}
	}
	if _, err := AccessibleText("T", []AccessibleRow{{Category: "C", Value: "123456789"}}, AccessibleRenderPolicy{MaxOutputBytes: 1}); err == nil || !strings.Contains(err.Error(), "chart.output_limit") {
		t.Fatalf("overflow error = %v", err)
	}
}

func TestAccessibleRowsValidateConditionalFormsAndEscapeReversibly(t *testing.T) {
	valid := []AccessibleRow{
		{Category: "Desktop", Value: "40"},
		{Series: "Revenue", Category: "Development", Value: "12"},
		{Series: "Latency", Category: "p50", Sample: "12", Value: "0.2"},
	}
	for _, row := range valid {
		if err := validateRows([]AccessibleRow{row}); err != nil {
			t.Fatalf("valid row rejected: %#v: %v", row, err)
		}
	}
	for _, row := range []AccessibleRow{{Value: "40"}, {Category: "Desktop"}, {Sample: "p50", Category: "Latency", Value: "12"}} {
		if err := validateRows([]AccessibleRow{row}); err == nil || !strings.Contains(err.Error(), "chart.accessibility_row_invalid") {
			t.Fatalf("invalid row accepted: %#v: %v", row, err)
		}
	}
	original := "Revenue|Development\n"
	encoded, err := EscapeAccessibleField(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnescapeAccessibleField(encoded)
	if err != nil || decoded != original {
		t.Fatalf("escape round trip = %q, %v", decoded, err)
	}
}

func TestAccessibleEscapingCoversEveryRowForm(t *testing.T) {
	rows := []AccessibleRow{
		{Category: "A|B", Value: "V%1"},
		{Series: "S\n", Category: "C\r", Value: "V\x7f"},
		{Series: "S", Category: "C", Sample: "p|", Value: "V\x01"},
	}
	out, err := AccessibleText("T", rows, AccessibleRenderPolicy{MaxOutputBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A%7CB|V%251", "S%0A|C%0D|V%7F", "S|C|p%7C|V%01"}
	if got := extractAccessibleRows(out); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("accessible rows = %#v, want %#v", got, want)
	}
	if _, err := EscapeAccessibleField(string([]byte{0xff})); err == nil || !strings.Contains(err.Error(), "chart.accessibility_field_invalid") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}

func TestAccessibleDataLimitFailsBeforeFirstByte(t *testing.T) {
	chart := templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := io.WriteString(out, strings.Repeat("x", 32))
		return err
	})
	var dst bytes.Buffer
	err := WithAccessibleData(chart, AccessibleData{Title: "T", Rows: []AccessibleRow{{Category: "C", Value: "1"}}}, AccessibleRenderPolicy{MaxOutputBytes: 8}).Render(context.Background(), &dst)
	if err == nil || !strings.Contains(err.Error(), "chart.output_limit") {
		t.Fatalf("overflow error = %v", err)
	}
	if dst.Len() != 0 {
		t.Fatalf("caller received %d bytes on overflow", dst.Len())
	}
}

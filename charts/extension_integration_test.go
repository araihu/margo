package charts

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	margo "github.com/araihu/margo"
)

func TestRootCompileDispatchesEveryV1ChartFamily(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "bar", body: readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml"), want: "Revenue|Development|12\x00Revenue|Production|18\x00Cost|Development|7\x00Cost|Production|9"},
		{name: "line", body: readChartFixtureForIntegration(t, "testdata/line/line-valid.yaml"), want: "Revenue|Development|12\x00Revenue|Production|18\x00Cost|Development|7\x00Cost|Production|9"},
		{name: "pie", body: readChartFixtureForIntegration(t, "testdata/pie/pie-valid.json"), want: "Desktop|40\x00Mobile|60"},
		{name: "doughnut", body: readChartFixtureForIntegration(t, "testdata/pie/doughnut-valid.json"), want: "Desktop|40\x00Mobile|60"},
		{name: "scatter", body: readChartFixtureForIntegration(t, "testdata/scatter/scatter-valid.yaml"), want: "Latency|p50|12\x00Latency|p95|18\x00Throughput|p50|30\x00Throughput|p95|42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, observed, err := renderThroughRoot(t, tc.body, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(extractAccessibleRows(string(out)), "\x00") != tc.want {
				t.Fatalf("rows = %#v", extractAccessibleRows(string(out)))
			}
			if observed != 1<<20 {
				t.Fatalf("observed policy = %d", observed)
			}
		})
	}
}

func TestRootDefaultEnablesChartControlWrapper(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, observed, err := renderThroughRoot(t, body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`data-goshtoso-chart-wrapper`,
		`data-goshtoso-chart-wrapper-mode="enabled"`,
		`data-goshtoso-chart-capability="static-svg"`,
		`data-goshtoso-chart-export-filename="revenue"`,
		`assets/js/controls/5/controls.js`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("default chart output missing %q", marker)
		}
	}
	if observed != 1<<20 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestRootCanDisableChartControlWrapper(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, observed, err := renderThroughRootWithExtension(t, body, 1<<20, Extension(WithControlWrapper(false)))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`data-goshtoso-chart-wrapper`,
		`assets/js/controls/5/controls.js`,
		`data-goshtoso-chart-export-filename`,
	} {
		if strings.Contains(markup, marker) {
			t.Fatalf("disabled chart output contains %q", marker)
		}
	}
	if !strings.Contains(markup, `data-margo-chart-data="v1"`) {
		t.Fatal("disabled chart output lost accessible data table")
	}
	if observed != 1<<20 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestChartControlWrapperConfigurationIsIdentityBound(t *testing.T) {
	enabled := Extension().Identity.ConfigurationHash
	disabled := Extension(WithControlWrapper(false)).Identity.ConfigurationHash
	if enabled == "" || disabled == "" || enabled == disabled {
		t.Fatalf("configuration hashes = enabled %q disabled %q", enabled, disabled)
	}
}

func TestRootPolicyOverflowWritesNoCallerBytes(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, observed, err := renderThroughRoot(t, body, 1)
	if err == nil || !strings.Contains(err.Error(), "chart.output_limit") {
		t.Fatalf("overflow error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("overflow returned %d bytes", len(out))
	}
	if observed != 1 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestRootCompilerMismatchFailsBeforeChartSession(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	compiler := margo.New(margo.WithHostPolicy(margo.Policy{OutputBytes: 1 << 20}), margo.WithExtension(Extension()))
	doc, err := compiler.Compile(context.Background(), margo.Source{Name: "bar.md", Content: []byte("```goshtosochart\n" + body + "\n```")})
	if err != nil {
		t.Fatal(err)
	}
	observed := int64(0)
	restore := setAccessiblePolicyObserverForTest(func(policy AccessibleRenderPolicy) { observed = policy.MaxOutputBytes })
	defer restore()
	mismatched := margo.New(margo.WithHostPolicy(margo.Policy{OutputBytes: 1}), margo.WithExtension(Extension()))
	_, err = mismatched.Render(context.Background(), doc)
	if err == nil || !strings.Contains(err.Error(), "compiler.document_config_mismatch") {
		t.Fatalf("mismatch error = %v", err)
	}
	if observed != 0 {
		t.Fatalf("chart session observed policy before mismatch: %d", observed)
	}
}

func TestChartSessionIsSafeForConcurrentRenders(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/line/line-valid.yaml")
	session, err := extensionFactory(margo.RenderContext{EffectivePolicy: margo.EffectivePolicy{OutputBytes: 1 << 20}})
	if err != nil {
		t.Fatal(err)
	}
	const renders = 8
	var wg sync.WaitGroup
	for i := 0; i < renders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out bytes.Buffer
			if err := session.Render(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: []byte(body)}, &out); err != nil {
				t.Errorf("concurrent render: %v", err)
			}
		}()
	}
	wg.Wait()
}

func renderThroughRoot(t *testing.T, body string, limit int64) ([]byte, int64, error) {
	return renderThroughRootWithExtension(t, body, limit, Extension())
}

func renderThroughRootWithExtension(t *testing.T, body string, limit int64, extension margo.ExtensionRegistration) ([]byte, int64, error) {
	t.Helper()
	var observed int64
	restore := setAccessiblePolicyObserverForTest(func(policy AccessibleRenderPolicy) { observed = policy.MaxOutputBytes })
	defer restore()
	compiler := margo.New(margo.WithHostPolicy(margo.Policy{OutputBytes: limit}), margo.WithExtension(extension))
	doc, err := compiler.Compile(context.Background(), margo.Source{Name: "chart.md", Content: []byte("```goshtosochart\n" + body + "\n```")})
	if err != nil {
		return nil, observed, err
	}
	result, err := compiler.Render(context.Background(), doc)
	if err != nil {
		return nil, observed, err
	}
	var out bytes.Buffer
	err = result.Content().Render(context.Background(), &out)
	return out.Bytes(), observed, err
}

func readChartFixtureForIntegration(t *testing.T, name string) string {
	t.Helper()
	body, err := readFixtureFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

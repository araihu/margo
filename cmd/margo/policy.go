package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
	"github.com/spf13/cobra"
)

const maxPolicyBytes = margo.MaxPolicyBytes

type policyTarget string

const (
	policyTargetHTML policyTarget = "html"
	policyTargetPDF  policyTarget = "pdf"
	policyTargetSite policyTarget = "site"
	policyTargetDeck policyTarget = "deck"
)

type loadedPolicy struct {
	Host            margo.Policy
	Digest          string
	AllowUnsafeHTML bool
}

type policyFlags struct {
	Path            string
	AllowUnsafeHTML bool
}

func (flags *policyFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&flags.Path, "policy", "", "trusted host policy JSON file")
	bindUnsafeHTMLFlag(command, &flags.AllowUnsafeHTML)
}

func bindUnsafeHTMLFlag(command *cobra.Command, target *bool) {
	command.Flags().BoolVar(target, "allow-unsafe-html", false, "allow arbitrary document HTML and iframe markup (unsafe; disabled by default)")
	// Keep the raw-HTML vocabulary available for scripts that already use it;
	// both spellings intentionally share one opt-in switch.
	command.Flags().BoolVar(target, "allow-raw-html", false, "alias for --allow-unsafe-html")
}

func (flags policyFlags) load(ctx context.Context, reader SourceReader) (*loadedPolicy, error) {
	if strings.TrimSpace(flags.Path) == "" {
		if flags.AllowUnsafeHTML {
			return &loadedPolicy{AllowUnsafeHTML: true}, nil
		}
		return nil, nil
	}
	if flags.Path == "-" {
		return nil, fmt.Errorf("cli.policy_invalid: --policy requires a file path, not stdin")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("cli.policy_read: policy reader is unavailable")
	}
	content, err := reader.ReadFile(flags.Path, maxPolicyBytes)
	if err != nil {
		return nil, fmt.Errorf("cli.policy_read: %w", err)
	}
	policy, err := parsePolicyDocument(content)
	if err != nil {
		return nil, err
	}
	policy.AllowUnsafeHTML = flags.AllowUnsafeHTML
	return &policy, nil
}

func parsePolicyDocument(input []byte) (loadedPolicy, error) {
	policy, err := margo.ParsePolicyJSON(input)
	if err != nil {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: %w", err)
	}
	canonical, err := json.Marshal(policy)
	if err != nil {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: %w", err)
	}
	hash := sha256.Sum256(canonical)
	return loadedPolicy{Host: policy, Digest: "sha256:" + hex.EncodeToString(hash[:])}, nil
}

func compilerForPolicy(policy *loadedPolicy, target policyTarget, chartOptions ...charts.Option) *margo.Compiler {
	if target == policyTargetDeck {
		// Full-page decks are static slide artifacts. Keep the chart SVG and
		// accessible data table, but do not ship browser-only chart controls.
		chartOptions = append(chartOptions, charts.WithDeckProjection(true), charts.WithControlWrapper(false))
	}
	options := make([]margo.Option, 0, 2)
	if policy != nil && policy.Digest != "" {
		options = append(options, margo.WithHostPolicy(policy.Host))
	}
	if policy != nil && policy.AllowUnsafeHTML {
		options = append(options, margo.WithUnsafeHTML())
	}
	return newCompilerWithChartOptions(chartOptions, options...)
}

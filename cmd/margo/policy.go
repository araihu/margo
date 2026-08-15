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
	Host   margo.Policy
	Digest string
}

type policyFlags struct{ Path string }

func (flags *policyFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&flags.Path, "policy", "", "trusted host policy JSON file")
}

func (flags policyFlags) load(ctx context.Context, reader SourceReader) (*loadedPolicy, error) {
	if strings.TrimSpace(flags.Path) == "" {
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

func compilerForPolicy(policy *loadedPolicy, _ policyTarget, chartOptions ...charts.Option) *margo.Compiler {
	if policy == nil {
		return newCompilerWithChartOptions(chartOptions)
	}
	return newCompilerWithChartOptions(chartOptions, margo.WithHostPolicy(policy.Host))
}

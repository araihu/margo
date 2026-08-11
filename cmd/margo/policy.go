package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	margo "github.com/araihu/margo"
	margoembed "github.com/araihu/margo/embed"
	"github.com/spf13/cobra"
)

const (
	policySchemaVersion = "margo-policy/v1"
	maxPolicyBytes      = 64 << 10
)

type policyTarget string

const (
	policyTargetHTML policyTarget = "html"
	policyTargetPDF  policyTarget = "pdf"
	policyTargetSite policyTarget = "site"
	policyTargetDeck policyTarget = "deck"
)

type policyProjections struct {
	HTML margoembed.Projection `json:"html"`
	PDF  margoembed.Projection `json:"pdf"`
	Site margoembed.Projection `json:"site"`
	Deck margoembed.Projection `json:"deck"`
}

type policyEmbedDocument struct {
	AllowedKinds   []margoembed.Kind         `json:"allowedKinds"`
	AllowedOrigins []string                  `json:"allowedOrigins"`
	IframeSandbox  []margoembed.SandboxToken `json:"iframeSandbox,omitempty"`
	ReferrerPolicy margoembed.ReferrerPolicy `json:"referrerPolicy,omitempty"`
	Projections    policyProjections         `json:"projections"`
}

type policyDocument struct {
	SchemaVersion string               `json:"schemaVersion"`
	RawHTML       margo.RawHTMLMode    `json:"rawHTML"`
	OutputBytes   int64                `json:"outputBytes,omitempty"`
	TrustedEmbeds *policyEmbedDocument `json:"trustedEmbeds,omitempty"`
}

type loadedPolicy struct {
	Host     margo.Policy
	Digest   string
	document policyDocument
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
	content, err := reader.ReadFile(flags.Path)
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
	if len(input) == 0 || len(input) > maxPolicyBytes {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: policy must contain 1 to %d bytes", maxPolicyBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	var document policyDocument
	if err := decoder.Decode(&document); err != nil {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: policy must contain one JSON object")
	}
	if document.SchemaVersion != policySchemaVersion {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: schemaVersion must be %q", policySchemaVersion)
	}
	if document.RawHTML == "" {
		document.RawHTML = margo.RawHTMLDeny
	}
	if document.RawHTML != margo.RawHTMLDeny && document.RawHTML != margo.RawHTMLSanitized {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: rawHTML must be deny or sanitized")
	}
	if document.OutputBytes == 0 {
		document.OutputBytes = margo.MaxOutputBytes
	}
	if document.OutputBytes < margo.MinOutputBytes || document.OutputBytes > margo.MaxOutputBytes {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: outputBytes must be between %d and %d", margo.MinOutputBytes, margo.MaxOutputBytes)
	}
	if document.TrustedEmbeds != nil {
		normalized, err := normalizePolicyEmbeds(*document.TrustedEmbeds)
		if err != nil {
			return loadedPolicy{}, err
		}
		document.TrustedEmbeds = &normalized
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return loadedPolicy{}, fmt.Errorf("cli.policy_invalid: %w", err)
	}
	hash := sha256.Sum256(canonical)
	return loadedPolicy{
		Host:   margo.Policy{RawHTML: document.RawHTML, OutputBytes: document.OutputBytes},
		Digest: "sha256:" + hex.EncodeToString(hash[:]), document: document,
	}, nil
}

func normalizePolicyEmbeds(document policyEmbedDocument) (policyEmbedDocument, error) {
	projections := []*margoembed.Projection{
		&document.Projections.HTML, &document.Projections.PDF,
		&document.Projections.Site, &document.Projections.Deck,
	}
	for _, projection := range projections {
		if *projection == "" {
			*projection = margoembed.ProjectionDeny
		}
		if *projection != margoembed.ProjectionDeny && *projection != margoembed.ProjectionStaticLink && *projection != margoembed.ProjectionInteractive {
			return policyEmbedDocument{}, fmt.Errorf("cli.policy_invalid: embed projection must be deny, static-link, or interactive")
		}
	}
	capabilityProjection := margoembed.ProjectionDeny
	for _, projection := range []margoembed.Projection{
		document.Projections.HTML, document.Projections.PDF, document.Projections.Site, document.Projections.Deck,
	} {
		if projection != margoembed.ProjectionDeny {
			capabilityProjection = projection
			break
		}
	}
	base := margoembed.Policy{
		Projection: capabilityProjection, AllowedKinds: document.AllowedKinds,
		AllowedOrigins: document.AllowedOrigins, IframeSandbox: document.IframeSandbox,
		ReferrerPolicy: document.ReferrerPolicy,
	}
	normalized, err := margoembed.NormalizePolicy(base)
	if err != nil {
		return policyEmbedDocument{}, fmt.Errorf("cli.policy_invalid: %w", err)
	}
	document.AllowedKinds = normalized.AllowedKinds
	document.AllowedOrigins = normalized.AllowedOrigins
	document.IframeSandbox = normalized.IframeSandbox
	document.ReferrerPolicy = normalized.ReferrerPolicy
	return document, nil
}

func (policy loadedPolicy) EmbedPolicy(target policyTarget) (margoembed.Policy, bool) {
	if policy.document.TrustedEmbeds == nil {
		return margoembed.Policy{}, false
	}
	document := policy.document.TrustedEmbeds
	projection := margoembed.ProjectionDeny
	switch target {
	case policyTargetHTML:
		projection = document.Projections.HTML
	case policyTargetPDF:
		projection = document.Projections.PDF
	case policyTargetSite:
		projection = document.Projections.Site
	case policyTargetDeck:
		projection = document.Projections.Deck
	default:
		return margoembed.Policy{}, false
	}
	result := margoembed.Policy{
		Projection: projection, AllowedKinds: append([]margoembed.Kind(nil), document.AllowedKinds...),
		AllowedOrigins: append([]string(nil), document.AllowedOrigins...),
		IframeSandbox:  append([]margoembed.SandboxToken(nil), document.IframeSandbox...),
		ReferrerPolicy: document.ReferrerPolicy,
	}
	return result, true
}

func (policy loadedPolicy) CheckEmbedPolicy() (margoembed.Policy, bool) {
	if policy.document.TrustedEmbeds == nil {
		return margoembed.Policy{}, false
	}
	targets := []policyTarget{policyTargetHTML, policyTargetSite, policyTargetPDF, policyTargetDeck}
	for _, target := range targets {
		candidate, _ := policy.EmbedPolicy(target)
		if candidate.Projection != margoembed.ProjectionDeny {
			candidate.Projection = margoembed.ProjectionStaticLink
			return candidate, true
		}
	}
	candidate, _ := policy.EmbedPolicy(policyTargetHTML)
	return candidate, true
}

func compilerForPolicy(policy *loadedPolicy, target policyTarget) *margo.Compiler {
	if policy == nil {
		return newCompiler()
	}
	options := []margo.Option{margo.WithHostPolicy(policy.Host)}
	if embedPolicy, ok := policy.EmbedPolicy(target); ok {
		options = append(options, margo.WithExtension(margoembed.Extension(embedPolicy)))
	}
	return newCompiler(options...)
}

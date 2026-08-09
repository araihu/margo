package margo

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/araihu/margo/internal/canonicaljson"
)

type HTMLFingerprint [32]byte

func (f HTMLFingerprint) String() string { return hex.EncodeToString(f[:]) }

type htmlRequirementFingerprint struct {
	ID           string              `json:"id"`
	Kind         HTMLRequirementKind `json:"kind"`
	LocalURL     string              `json:"localURL,omitempty"`
	Integrity    string              `json:"integrity,omitempty"`
	LoadAfter    []string            `json:"loadAfter,omitempty"`
	InlinePath   string              `json:"inlinePath,omitempty"`
	InlineType   string              `json:"inlineType,omitempty"`
	InlineSHA256 string              `json:"inlineSHA256,omitempty"`
}

func htmlFingerprint(fragment []byte, metadata HTMLMetadata, requirements HTMLRequirements, config htmlConfig) (HTMLFingerprint, error) {
	fragmentDigest := sha256.Sum256(fragment)
	list := requirements.List()
	projection := make([]htmlRequirementFingerprint, len(list))
	for index, requirement := range list {
		projection[index] = htmlRequirementFingerprint{
			ID: requirement.ID, Kind: requirement.Kind, LocalURL: requirement.LocalURL,
			Integrity: requirement.Integrity, LoadAfter: append([]string(nil), requirement.LoadAfter...),
			InlinePath: requirement.Inline.Path, InlineType: requirement.Inline.MediaType, InlineSHA256: requirement.Inline.SHA256,
		}
	}
	encoded, err := canonicaljson.Marshal(struct {
		FragmentSHA256 string                       `json:"fragmentSHA256"`
		Metadata       HTMLMetadata                 `json:"metadata"`
		Requirements   []htmlRequirementFingerprint `json:"requirements,omitempty"`
		Header         bool                         `json:"header"`
	}{
		FragmentSHA256: hex.EncodeToString(fragmentDigest[:]),
		Metadata:       metadata.clone(),
		Requirements:   projection,
		Header:         config.header,
	})
	if err != nil {
		return HTMLFingerprint{}, err
	}
	preimage := append([]byte("margo/html/v1\n"), encoded...)
	return HTMLFingerprint(sha256.Sum256(preimage)), nil
}

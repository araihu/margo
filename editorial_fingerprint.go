package margo

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/araihu/margo/internal/canonicaljson"
)

type EditorialFingerprint [32]byte

func (f EditorialFingerprint) String() string { return hex.EncodeToString(f[:]) }

type editorialRequirementFingerprint struct {
	ID           string              `json:"id"`
	Kind         HTMLRequirementKind `json:"kind"`
	LocalURL     string              `json:"localURL,omitempty"`
	Integrity    string              `json:"integrity,omitempty"`
	LoadAfter    []string            `json:"loadAfter,omitempty"`
	InlinePath   string              `json:"inlinePath,omitempty"`
	InlineType   string              `json:"inlineType,omitempty"`
	InlineSHA256 string              `json:"inlineSHA256,omitempty"`
}

func editorialFingerprint(fragment []byte, metadata EditorialMetadata, requirements HTMLRequirements, config editorialConfig) (EditorialFingerprint, error) {
	fragmentDigest := sha256.Sum256(fragment)
	list := requirements.List()
	projection := make([]editorialRequirementFingerprint, len(list))
	for index, requirement := range list {
		projection[index] = editorialRequirementFingerprint{
			ID: requirement.ID, Kind: requirement.Kind, LocalURL: requirement.LocalURL,
			Integrity: requirement.Integrity, LoadAfter: append([]string(nil), requirement.LoadAfter...),
			InlinePath: requirement.Inline.Path, InlineType: requirement.Inline.MediaType, InlineSHA256: requirement.Inline.SHA256,
		}
	}
	encoded, err := canonicaljson.Marshal(struct {
		FragmentSHA256 string                            `json:"fragmentSHA256"`
		Metadata       EditorialMetadata                 `json:"metadata"`
		Requirements   []editorialRequirementFingerprint `json:"requirements,omitempty"`
		Header         bool                              `json:"header"`
	}{
		FragmentSHA256: hex.EncodeToString(fragmentDigest[:]),
		Metadata:       metadata.clone(),
		Requirements:   projection,
		Header:         config.header,
	})
	if err != nil {
		return EditorialFingerprint{}, err
	}
	preimage := append([]byte("margo/editorial/v1\n"), encoded...)
	return EditorialFingerprint(sha256.Sum256(preimage)), nil
}

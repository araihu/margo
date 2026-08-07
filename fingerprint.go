package margo

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/araihu/margo/internal/canonicaljson"
)

// CompilerConfigFingerprint identifies the frozen compiler configuration.
type CompilerConfigFingerprint [32]byte

// DocumentFingerprint identifies the immutable compiled meaning.
type DocumentFingerprint [32]byte

func (f CompilerConfigFingerprint) String() string { return hex.EncodeToString(f[:]) }
func (f DocumentFingerprint) String() string       { return hex.EncodeToString(f[:]) }

func compilerConfigFingerprint(values map[string]any) CompilerConfigFingerprint {
	bytes, err := canonicaljson.Marshal(values)
	if err != nil {
		panic(err)
	}
	preimage := append([]byte("margo/compiler-config/v1\n"), bytes...)
	return CompilerConfigFingerprint(sha256.Sum256(preimage))
}

func documentFingerprint(source Source, compiler CompilerConfigFingerprint, values map[string]any) DocumentFingerprint {
	sourceHash := sha256.Sum256(source.Content)
	preimage := map[string]any{
		"baseURL":                   source.BaseURL,
		"compilerConfigFingerprint": compiler.String(),
		"sourceName":                source.Name,
		"sourceSHA256":              hex.EncodeToString(sourceHash[:]),
		"values":                    values,
	}
	bytes, err := canonicaljson.Marshal(preimage)
	if err != nil {
		panic(err)
	}
	data := append([]byte("margo/document/v1\n"), bytes...)
	return DocumentFingerprint(sha256.Sum256(data))
}

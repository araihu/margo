package mermaid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/araihu/margo/internal/canonicaljson"
)

const (
	RuntimeVersion        = "11.16.1"
	RuntimeDigest         = "sha384:4ebed2d056672dc504310c8a5be4d28abe2b2a08c0c11487650801f9528cb8cb2ad6faf66bbb1ae9db2aeff023fd414f"
	ProfileFingerprintHex = "bfe4c79b9ccb911c2511c5d24fe14458d148cd64e4bcd5faab97acc84b6cfd1a"
)

type TaskInput struct {
	SourceSHA256       [32]byte `json:"sourceSHA256"`
	BlockOrdinal       uint32   `json:"blockOrdinal"`
	RuntimeDigest      string   `json:"runtimeDigest"`
	ProfileFingerprint [32]byte `json:"profileFingerprint"`
}

// TaskDescriptor freezes the source and configuration identities known before
// browser execution. It deliberately carries no generated SVG or live state.
type TaskDescriptor struct {
	Kind              string    `json:"kind"`
	BlockID           string    `json:"blockID"`
	ConfigurationHash [32]byte  `json:"configurationHash"`
	Input             TaskInput `json:"input"`
}

type strictConfigurationIdentity struct {
	SchemaVersion     string `json:"schemaVersion"`
	Mode              string `json:"mode"`
	SecurityLevel     string `json:"securityLevel"`
	StartOnLoad       bool   `json:"startOnLoad"`
	HTMLLabels        bool   `json:"htmlLabels"`
	DiagramHTMLLabels bool   `json:"diagramHTMLLabels"`
	ThemeCSS          string `json:"themeCSS"`
	DeterministicIDs  bool   `json:"deterministicIds"`
	Look              string `json:"look"`
	Layout            string `json:"layout"`
	CallbacksEnabled  bool   `json:"callbacksEnabled"`
	ExternalIcons     bool   `json:"externalIcons"`
}

var profileFingerprint = mustDigest32(ProfileFingerprintHex)

// Compile runs fail-closed preflight before constructing the immutable task.
func Compile(source []byte, blockOrdinal uint32) (TaskDescriptor, error) {
	if err := Preflight(source); err != nil {
		return TaskDescriptor{}, err
	}
	sourceDigest := sha256.Sum256(source)
	return TaskDescriptor{
		Kind:              "mermaid",
		BlockID:           fmt.Sprintf("mermaid:%08d:%s", blockOrdinal, hex.EncodeToString(sourceDigest[:])),
		ConfigurationHash: StrictConfigurationHash(),
		Input: TaskInput{
			SourceSHA256:       sourceDigest,
			BlockOrdinal:       blockOrdinal,
			RuntimeDigest:      RuntimeDigest,
			ProfileFingerprint: profileFingerprint,
		},
	}, nil
}

// StrictConfigurationHash identifies the one literal v0.0.1 configuration.
func StrictConfigurationHash() [32]byte {
	identity := strictConfigurationIdentity{
		SchemaVersion:     "margo/mermaid-configuration/v1",
		Mode:              "strict",
		SecurityLevel:     "strict",
		StartOnLoad:       false,
		HTMLLabels:        false,
		DiagramHTMLLabels: false,
		ThemeCSS:          "",
		DeterministicIDs:  true,
		Look:              "classic",
		Layout:            "dagre",
		CallbacksEnabled:  false,
		ExternalIcons:     false,
	}
	encoded, err := canonicaljson.Marshal(identity)
	if err != nil {
		panic(err)
	}
	preimage := append([]byte("margo/mermaid-configuration/v1\n"), encoded...)
	return sha256.Sum256(preimage)
}

func mustDigest32(value string) [32]byte {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		panic("invalid fixed Mermaid digest")
	}
	var result [32]byte
	copy(result[:], decoded)
	return result
}

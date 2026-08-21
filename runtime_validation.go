package margo

import (
	"encoding/hex"
	"math"
)

const RuntimeProtocolV2 = "margo-runtime/v2"

// RuntimeValidationRequest is the profile-neutral request bound to a v2
// descriptor. Deck owns the profile registry and derives the font digest;
// margo validates the wire shape and equality constraints.
type RuntimeValidationRequest struct {
	ViewportWidth            uint    `json:"viewportWidth"`
	ViewportHeight           uint    `json:"viewportHeight"`
	DeviceScaleFactor        float64 `json:"deviceScaleFactor"`
	Zoom                     float64 `json:"zoom"`
	BrowserProfile           string  `json:"browserProfile"`
	ExpectedFontBundleDigest string  `json:"expectedFontBundleDigest"`
}

func (request RuntimeValidationRequest) Validate() error {
	if request.ViewportWidth == 0 || request.ViewportHeight == 0 ||
		math.IsNaN(request.DeviceScaleFactor) || math.IsInf(request.DeviceScaleFactor, 0) || request.DeviceScaleFactor <= 0 ||
		math.IsNaN(request.Zoom) || math.IsInf(request.Zoom, 0) || request.Zoom <= 0 ||
		request.BrowserProfile == "" || !validRuntimeDigest(request.ExpectedFontBundleDigest) {
		return runtimeDiagnostic("runtime.validation_request_invalid", "runtime validation request is incomplete or invalid")
	}
	return nil
}

// RuntimeValidationIdentity records values observed by the validator rather
// than caller assertions.
type RuntimeValidationIdentity struct {
	BrowserProfile   string `json:"browserProfile"`
	EngineName       string `json:"engineName"`
	EngineVersion    string `json:"engineVersion"`
	PlatformProfile  string `json:"platformProfile"`
	FontBundleDigest string `json:"fontBundleDigest"`
}

func (identity RuntimeValidationIdentity) Validate() error {
	if identity.BrowserProfile == "" || identity.EngineName == "" || identity.EngineVersion == "" || identity.PlatformProfile == "" || !validRuntimeDigest(identity.FontBundleDigest) {
		return runtimeDiagnostic("runtime.validation_identity_invalid", "runtime validation identity is incomplete or invalid")
	}
	return nil
}

func validRuntimeDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && runtimeDigestPattern.MatchString(value)
}

// Package mathfonts shares the canonical math font-bundle identity with decks.
package mathfonts

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
)

var facePattern = regexp.MustCompile(`@font-face\{[^}]*font-family:([^;]+);font-style:([^;]+);font-weight:([^;]+);src:url\(data:font/woff2;base64,([^)]*)\)[^}]*\}`)

// Digest hashes font metadata and decoded WOFF2 bytes in stylesheet face order.
func Digest(css []byte) (string, error) {
	faces := facePattern.FindAllSubmatch(css, -1)
	if len(faces) != 20 {
		return "", fmt.Errorf("math.font_unavailable: expected 20 bundled faces")
	}
	h := sha256.New()
	h.Write([]byte("margo-math-fonts/v1\x00"))
	for _, face := range faces {
		for index, value := range face[1:4] {
			if index == 0 {
				value = bytes.Trim(value, `"'`)
			}
			h.Write(value)
			h.Write([]byte{0})
		}
		data, err := base64.StdEncoding.DecodeString(string(face[4]))
		if err != nil {
			return "", err
		}
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(data)))
		h.Write(length[:])
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Combine keeps the existing theme identity and binds the additional math faces.
func Combine(theme, math string) string {
	sum := sha256.Sum256([]byte("margo-font-bundle/math-v1\x00" + theme + "\x00" + math))
	return hex.EncodeToString(sum[:])
}

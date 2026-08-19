package deck

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/araihu/margo"
)

func bundledFontFaces(theme margo.ThemeName) ([]FontFaceAsset, error) {
	families := []string{"Margo Sans", "Margo Mono"}
	if theme == margo.ThemeMinimal {
		families = []string{"Margo Sans", "Margo Serif", "Margo Mono"}
	}
	weights := map[string][]int{
		"Margo Sans":  {400, 600, 700, 800},
		"Margo Serif": {400, 600, 700},
		"Margo Mono":  {400, 600},
	}
	faces := make([]FontFaceAsset, 0)
	for _, family := range families {
		for _, weight := range weights[family] {
			bytes, err := embeddedDeckFont(fontAssetFile(family))
			if err != nil {
				return nil, err
			}
			faces = append(faces, FontFaceAsset{
				Family: family,
				Weight: weight,
				Bytes:  bytes,
			})
		}
	}
	return faces, nil
}

func bundledFontDigest(theme margo.ThemeName) (string, error) {
	faces, err := bundledFontFaces(theme)
	if err != nil {
		return "", err
	}
	return FontBundleDigestV1(faces)
}

func fontAssetFile(family string) string {
	switch family {
	case "Margo Serif":
		return "margo-serif.woff2"
	case "Margo Mono":
		return "margo-mono.woff2"
	default:
		return "margo-sans.woff2"
	}
}

// FontFaceAsset is one immutable versioned WOFF2 face used by a deck theme.
// Bytes are copied before hashing so callers cannot mutate a digest input.
type FontFaceAsset struct {
	Family string
	Weight int
	Bytes  []byte
}

var fontFamilyOrder = map[string]int{
	"Margo Sans":  0,
	"Margo Serif": 1,
	"Margo Mono":  2,
}

// FontBundleDigestV1 hashes the canonical margo-font-bundle/v1 preimage.
// Callers must provide the required faces in theme-row order and ascending
// weight order; accepting a different order would make two valid locks diverge.
func FontBundleDigestV1(faces []FontFaceAsset) (string, error) {
	if len(faces) == 0 {
		return "", fmt.Errorf("deck.font_bundle_invalid: at least one face is required")
	}
	seen := make(map[string]struct{}, len(faces))
	previous := FontFaceAsset{}
	for index, face := range faces {
		if face.Family == "" || face.Weight <= 0 || face.Weight > 1000 || len(face.Bytes) == 0 {
			return "", fmt.Errorf("deck.font_bundle_invalid: face %d is incomplete", index)
		}
		key := fmt.Sprintf("%s\x00%d", face.Family, face.Weight)
		if _, exists := seen[key]; exists {
			return "", fmt.Errorf("deck.font_bundle_invalid: duplicate face %s", key)
		}
		seen[key] = struct{}{}
		if index > 0 && compareFontFace(previous, face) >= 0 {
			return "", fmt.Errorf("deck.font_bundle_invalid: faces are not in canonical order")
		}
		previous = face
	}
	var preimage bytes.Buffer
	_, _ = preimage.WriteString("margo-font-bundle/v1")
	_ = preimage.WriteByte(0)
	for _, face := range faces {
		_, _ = preimage.WriteString(face.Family)
		_ = preimage.WriteByte(0)
		_, _ = preimage.WriteString(fmt.Sprintf("%d", face.Weight))
		_ = preimage.WriteByte(0)
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(face.Bytes)))
		_, _ = preimage.Write(length[:])
		_, _ = preimage.Write(append([]byte(nil), face.Bytes...))
	}
	digest := sha256.Sum256(preimage.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

func compareFontFace(left, right FontFaceAsset) int {
	leftRank, leftKnown := fontFamilyOrder[left.Family]
	rightRank, rightKnown := fontFamilyOrder[right.Family]
	if leftKnown && rightKnown && leftRank != rightRank {
		return compareInt(leftRank, rightRank)
	}
	if left.Family != right.Family {
		families := []string{left.Family, right.Family}
		sort.Strings(families)
		if families[0] == left.Family {
			return -1
		}
		return 1
	}
	return compareInt(left.Weight, right.Weight)
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

// Package canonicaljson provides the deterministic JSON byte routine used by
// Margo identity preimages. Encoder.Encode is used deliberately, then its one
// required terminal LF is removed for canonBytes; line-delimited callers add
// exactly one LF themselves.
package canonicaljson

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Marshal encodes v with HTML escaping disabled, sorted object keys, and no
// terminal newline. encoding/json rejects NaN and infinities before bytes are
// returned.
func Marshal(v any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	encoded := buffer.Bytes()
	if len(encoded) == 0 || encoded[len(encoded)-1] != '\n' {
		return nil, fmt.Errorf("canonicaljson: encoder omitted terminal LF")
	}
	return append([]byte(nil), encoded[:len(encoded)-1]...), nil
}

// Line encodes one canonical record and appends exactly one line-feed.
func Line(v any) ([]byte, error) {
	encoded, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

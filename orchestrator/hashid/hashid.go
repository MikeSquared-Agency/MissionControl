// Package hashid provides deterministic, content-addressable ID generation.
// IDs are derived from SHA-256 hashes of the input content, truncated to
// 10 hex characters. This makes IDs stable: the same content always produces
// the same ID.
package hashid

import (
	"crypto/sha256"
	"fmt"
)

// Generate produces a deterministic 10-character hex ID from the given
// content parts. Parts are length-prefixed before hashing to avoid
// ambiguous encodings (e.g., ["a|b"] vs ["a", "b"]).
func Generate(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		// Length-prefix each part to make encoding unambiguous
		fmt.Fprintf(h, "%d:%s", len(p), p)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:5]) // 5 bytes = 10 hex chars
}

// Package hashid provides deterministic, content-addressable ID generation.
// IDs are derived from SHA-256 hashes of the input content, truncated to
// 10 hex characters. This makes IDs stable: the same content always produces
// the same ID.
package hashid

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Generate produces a deterministic 10-character hex ID from the given
// content parts. Parts are joined with "|" before hashing.
func Generate(parts ...string) string {
	content := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash[:5]) // 5 bytes = 10 hex chars
}

package scaffold

import "fmt"

const (
	// DefaultCategory is the category used when no arguments are provided.
	DefaultCategory = "root"
	// DefaultName is the scaffold name used when no arguments are provided.
	DefaultName = "default"
)

// KnownCategories lists all recognized scaffold categories.
var KnownCategories = []string{"root", "project", "feature"}

// ShardPath computes the sharding file path from category and name
// using FNV-1a 32-bit hash, reduced to base36 4-character encoding.
//
// Algorithm:
//  1. key = category + "/" + name
//  2. FNV-1a 32-bit hash (offset_basis=2166136261, prime=16777619)
//  3. reduced = hash % 1679616 (36^4)
//  4. base36 encode to 4 chars (0-padded)
//  5. path = "catalog/scaffolds/{c0}/{c1}/{c2}/{c3}.yaml"
func ShardPath(category, name string) string {
	key := category + "/" + name

	// FNV-1a 32-bit hash
	const offsetBasis = uint32(2166136261)
	const prime = uint32(16777619)

	hash := offsetBasis
	for i := range len(key) {
		hash ^= uint32(key[i])
		hash *= prime
	}

	// Reduce to 36^4 space
	reduced := hash % 1679616

	// Base36 encode (4 chars, 0-padded, big-endian)
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	encoded := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		encoded[i] = chars[reduced%36]
		reduced /= 36
	}

	return fmt.Sprintf("catalog/scaffolds/%c/%c/%c/%c.yaml",
		encoded[0], encoded[1], encoded[2], encoded[3])
}

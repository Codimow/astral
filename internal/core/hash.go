package core

import (
	"encoding/hex"
	"io"

	"github.com/zeebo/blake3"
)

// Hash represents a Blake3 hash value
type Hash [32]byte

// String returns the hexadecimal representation of the hash
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Short returns the first 7 characters of the hash (like git)
func (h Hash) Short() string {
	return h.String()[:7]
}

// HashBytes computes the Blake3 hash of a byte slice
func HashBytes(data []byte) Hash {
	return blake3.Sum256(data)
}

// HashReader computes the Blake3 hash of data from an io.Reader
func HashReader(r io.Reader) (Hash, error) {
	hasher := blake3.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return Hash{}, err
	}

	var hash Hash
	copy(hash[:], hasher.Sum(nil))
	return hash, nil
}

// ParseHash parses a hex string into a Hash
func ParseHash(s string) (Hash, error) {
	var hash Hash
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return hash, err
	}
	if len(bytes) != 32 {
		return hash, ErrInvalidHash
	}
	copy(hash[:], bytes)
	return hash, nil
}

// IsZero returns true if the hash is all zeros
func (h Hash) IsZero() bool {
	return h == Hash{}
}

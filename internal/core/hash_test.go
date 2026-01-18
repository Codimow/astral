package core

import (
	"testing"
)

func TestHashBytes(t *testing.T) {
	data := []byte("hello world")
	hash := HashBytes(data)

	if hash.IsZero() {
		t.Error("expected non-zero hash")
	}

	// Same data should produce same hash
	hash2 := HashBytes(data)
	if hash != hash2 {
		t.Error("same data should produce same hash")
	}

	// Different data should produce different hash
	hash3 := HashBytes([]byte("goodbye world"))
	if hash == hash3 {
		t.Error("different data should produce different hash")
	}
}

func TestHashShort(t *testing.T) {
	data := []byte("test")
	hash := HashBytes(data)

	short := hash.Short()
	if len(short) != 7 {
		t.Errorf("expected short hash length 7, got %d", len(short))
	}

	// Short should be prefix of full hash
	full := hash.String()
	if full[:7] != short {
		t.Error("short hash should be prefix of full hash")
	}
}

func TestParseHash(t *testing.T) {
	original := HashBytes([]byte("test"))
	hashStr := original.String()

	parsed, err := ParseHash(hashStr)
	if err != nil {
		t.Fatalf("failed to parse hash: %v", err)
	}

	if parsed != original {
		t.Error("parsed hash should equal original")
	}

	// Test invalid hash
	_, err = ParseHash("invalid")
	if err == nil {
		t.Error("expected error for invalid hash")
	}

	// Test wrong length
	_, err = ParseHash("abc123")
	if err == nil {
		t.Error("expected error for wrong length hash")
	}
}

func BenchmarkHashBytes(b *testing.B) {
	data := make([]byte, 1024*1024) // 1 MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashBytes(data)
	}
}

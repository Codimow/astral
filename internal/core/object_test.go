package core

import (
	"testing"
	"time"
)

func TestEncodeDecodeCommit(t *testing.T) {
	original := &Commit{
		Tree:      HashBytes([]byte("tree")),
		Parent:    HashBytes([]byte("parent")),
		Author:    "Test Author",
		Email:     "test@example.com",
		Timestamp: time.Unix(1234567890, 0),
		Message:   "Test commit message",
	}

	// Encode
	data := EncodeCommit(original)
	if len(data) == 0 {
		t.Fatal("encoded data is empty")
	}

	// Decode
	decoded, err := DecodeCommit(data)
	if err != nil {
		t.Fatalf("failed to decode commit: %v", err)
	}

	// Verify fields
	if decoded.Tree != original.Tree {
		t.Error("tree hash mismatch")
	}
	if decoded.Parent != original.Parent {
		t.Error("parent hash mismatch")
	}
	if decoded.Author != original.Author {
		t.Error("author mismatch")
	}
	if decoded.Email != original.Email {
		t.Error("email mismatch")
	}
	if decoded.Timestamp.Unix() != original.Timestamp.Unix() {
		t.Error("timestamp mismatch")
	}
	if decoded.Message != original.Message {
		t.Error("message mismatch")
	}
}

func TestEncodeDecodeCommitNoParent(t *testing.T) {
	original := &Commit{
		Tree:      HashBytes([]byte("tree")),
		Parent:    Hash{}, // Zero hash (no parent)
		Author:    "Test Author",
		Email:     "test@example.com",
		Timestamp: time.Now(),
		Message:   "Initial commit",
	}

	data := EncodeCommit(original)
	decoded, err := DecodeCommit(data)
	if err != nil {
		t.Fatalf("failed to decode commit: %v", err)
	}

	if !decoded.Parent.IsZero() {
		t.Error("expected zero parent hash")
	}
}

func TestEncodeDecodeTree(t *testing.T) {
	original := &Tree{
		Entries: []TreeEntry{
			{Mode: 0100644, Name: "file1.txt", Hash: HashBytes([]byte("content1"))},
			{Mode: 0100755, Name: "script.sh", Hash: HashBytes([]byte("content2"))},
			{Mode: 0100644, Name: "file2.md", Hash: HashBytes([]byte("content3"))},
		},
	}

	// Encode
	data := EncodeTree(original)
	if len(data) == 0 {
		t.Fatal("encoded tree is empty")
	}

	// Decode
	decoded, err := DecodeTree(data)
	if err != nil {
		t.Fatalf("failed to decode tree: %v", err)
	}

	// Verify entries
	if len(decoded.Entries) != len(original.Entries) {
		t.Fatalf("expected %d entries, got %d", len(original.Entries), len(decoded.Entries))
	}

	for i, entry := range decoded.Entries {
		orig := original.Entries[i]
		if entry.Mode != orig.Mode {
			t.Errorf("entry %d: mode mismatch", i)
		}
		if entry.Name != orig.Name {
			t.Errorf("entry %d: name mismatch", i)
		}
		if entry.Hash != orig.Hash {
			t.Errorf("entry %d: hash mismatch", i)
		}
	}
}

func TestEmptyTree(t *testing.T) {
	tree := &Tree{Entries: []TreeEntry{}}

	data := EncodeTree(tree)
	decoded, err := DecodeTree(data)
	if err != nil {
		t.Fatalf("failed to decode empty tree: %v", err)
	}

	if len(decoded.Entries) != 0 {
		t.Errorf("expected empty tree, got %d entries", len(decoded.Entries))
	}
}

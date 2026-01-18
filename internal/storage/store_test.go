package storage

import (
	"os"
	"testing"
	"time"

	"github.com/codimo/astral/internal/core"
)

func TestStorePutGet(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)

	// Store blob
	data := []byte("hello world")
	hash, err := store.PutBlob(data)
	if err != nil {
		t.Fatalf("failed to put blob: %v", err)
	}

	// Retrieve blob
	obj, err := store.Get(hash)
	if err != nil {
		t.Fatalf("failed to get object: %v", err)
	}

	if obj.Type != core.ObjectTypeBlob {
		t.Errorf("expected blob, got %s", obj.Type)
	}

	if string(obj.Data) != string(data) {
		t.Error("data mismatch")
	}
}

func TestStoreDeduplication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)

	data := []byte("duplicate content")

	// Store same data twice
	hash1, err := store.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}

	hash2, err := store.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}

	// Should produce same hash
	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}

	// Should only create one file
	path := store.objectPath(hash1)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("object file should exist")
	}
}

func TestStorePutCommit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)

	// Create commit
	commit := &core.Commit{
		Tree:      core.HashBytes([]byte("tree")),
		Parent:    core.Hash{},
		Author:    "Test",
		Email:     "test@test.com",
		Timestamp: time.Now(),
		Message:   "Test commit",
	}

	// Store
	hash, err := store.PutCommit(commit)
	if err != nil {
		t.Fatalf("failed to put commit: %v", err)
	}

	// Retrieve
	retrieved, err := store.GetCommit(hash)
	if err != nil {
		t.Fatalf("failed to get commit: %v", err)
	}

	if retrieved.Message != commit.Message {
		t.Error("commit message mismatch")
	}
	if retrieved.Author != commit.Author {
		t.Error("author mismatch")
	}
}

func TestStoreCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)

	data := []byte("cached content")
	hash, err := store.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}

	// First get - from disk
	obj1, err := store.Get(hash)
	if err != nil {
		t.Fatal(err)
	}

	// Second get - from cache
	obj2, err := store.Get(hash)
	if err != nil {
		t.Fatal(err)
	}

	if string(obj1.Data) != string(obj2.Data) {
		t.Error("cache returned different data")
	}
}

func TestStoreNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)

	// Try to get non-existent object
	fakeHash := core.HashBytes([]byte("nonexistent"))
	_, err = store.Get(fakeHash)
	if err != core.ErrObjectNotFound {
		t.Errorf("expected ErrObjectNotFound, got %v", err)
	}
}

func BenchmarkStorePut(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "astral-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)
	data := make([]byte, 1024) // 1 KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data[0] = byte(i) // Make each blob unique
		store.PutBlob(data)
	}
}

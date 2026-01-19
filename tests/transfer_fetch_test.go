package tests

import (
	"errors"
	"testing"

	"github.com/codimo/astral/internal/transfer"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

type mockFetcher struct {
	objects map[core.Hash]*core.Object
}

func (m *mockFetcher) FetchObject(hash core.Hash) (*core.Object, error) {
	obj, ok := m.objects[hash]
	if !ok {
		return nil, errors.New("object not found")
	}
	return obj, nil
}

func TestFetch(t *testing.T) {
	// Setup store
	tmpDir := t.TempDir()
	store := storage.NewStore(tmpDir)

	// Helper to compute object hash (type + space + data)
	computeHash := func(t core.ObjectType, data []byte) core.Hash {
		content := append([]byte(string(t)+" "), data...)
		return core.HashBytes(content)
	}

	// Create some objects for "remote"
	blobData := []byte("hello world")
	blobHash := computeHash(core.ObjectTypeBlob, blobData)
	blobObj := &core.Object{Type: core.ObjectTypeBlob, Data: blobData}

	tree := &core.Tree{
		Entries: []core.TreeEntry{
			{Name: "file.txt", Hash: blobHash, Mode: 0100644},
		},
	}
	treeData := core.EncodeTree(tree)
	treeHash := computeHash(core.ObjectTypeTree, treeData)
	treeObj := &core.Object{Type: core.ObjectTypeTree, Data: treeData}

	commit := &core.Commit{
		Tree:    treeHash,
		Message: "Initial commit",
	}
	commitData := core.EncodeCommit(commit)
	commitHash := computeHash(core.ObjectTypeCommit, commitData)
	commitObj := &core.Object{Type: core.ObjectTypeCommit, Data: commitData}

	// Mock client
	client := &mockFetcher{
		objects: map[core.Hash]*core.Object{
			blobHash:   blobObj,
			treeHash:   treeObj,
			commitHash: commitObj,
		},
	}

	// Test Fetch
	tips := []core.Hash{commitHash}
	if err := transfer.Fetch(store, client, tips); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Verify objects in store
	if _, err := store.Get(commitHash); err != nil {
		t.Errorf("Commit not found in store")
	}
	if _, err := store.Get(treeHash); err != nil {
		t.Errorf("Tree not found in store")
	}
	if _, err := store.Get(blobHash); err != nil {
		t.Errorf("Blob not found in store")
	}
}

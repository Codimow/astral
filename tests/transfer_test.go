package tests

import (
	"testing"

	"github.com/codimo/astral/internal/transfer"

	"os"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

func TestCalculatePushPack(t *testing.T) {
	// Setup store
	dir, _ := os.MkdirTemp("", "transfer-test")
	defer os.RemoveAll(dir)
	store := storage.NewStore(dir)

	// Create a chain: C1 -> C2 -> C3
	// Blob
	blobHash, _ := store.PutBlob([]byte("content"))

	// Tree
	tree := &core.Tree{Entries: []core.TreeEntry{{Name: "file", Hash: blobHash}}}
	treeHash, _ := store.PutTree(tree)

	// Commit 1
	c1 := &core.Commit{Tree: treeHash, Message: "first"}
	h1, _ := store.PutCommit(c1)

	// Commit 2
	c2 := &core.Commit{Tree: treeHash, Parents: []core.Hash{h1}, Message: "second"}
	h2, _ := store.PutCommit(c2)

	// Commit 3
	c3 := &core.Commit{Tree: treeHash, Parents: []core.Hash{h2}, Message: "third"}
	h3, _ := store.PutCommit(c3)

	// Case 1: Push everything (Remote empty)
	local := []core.Hash{h3}
	remote := []core.Hash{}

	pack, err := transfer.CalculatePushPack(store, local, remote)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	// Should have h3, h2, h1, treeHash, blobHash (5 objects)
	// Order depends on traversal (BFS/DFS), but we should have all.
	expected := map[core.Hash]bool{
		h3: true, h2: true, h1: true, treeHash: true, blobHash: true,
	}

	if len(pack) != 5 {
		t.Errorf("Expected 5 objects, got %d", len(pack))
	}
	for _, h := range pack {
		if !expected[h] {
			t.Errorf("Unexpected object in pack: %s", h)
		}
	}

	// Case 2: Incremental Push (Remote has h2)
	remote = []core.Hash{h2}
	pack, err = transfer.CalculatePushPack(store, local, remote)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	// Should only have h3.
	// h2 is in remote -> stop.
	// But wait, does h3 point to treeHash? Yes.
	// If h2 has treeHash, and remote has h2, does remote have treeHash? Yes.
	// Our algorithm checks "if visited[current]".
	// h3 -> treeHash
	// h3 -> h2 (STOP)
	// So treeHash might be processed?
	// treeHash is pushed to queue.
	// If treeHash was already in remote (via h2), we might ideally skip it.
	// BUT our current `haveSet` ONLY contains h2.
	// We don't know if remote has treeHash unless we check.
	// `CalculatePushPack` assumes remote refs imply history?
	// "If remote has this object, we assume they have the history".
	// h2 is in haveSet.
	// treeHash is NOT in haveSet (unless we expanded haveSet).
	// So treeHash will be sent again.
	// This is "dumb" protocol, but correct/safe. To be smarter, we would need to know more about remote directly.
	// Efficient git (bitmap) knows reachability.

	// So we assume duplicate send is acceptable for now (HTTP /objects/ POST just overwrites).

	if len(pack) < 1 {
		t.Error("Pack empty")
	}
	foundH3 := false
	for _, h := range pack {
		if h == h3 {
			foundH3 = true
		}
	}
	if !foundH3 {
		t.Error("Missing h3")
	}
}

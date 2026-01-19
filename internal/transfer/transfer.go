package transfer

import (
	"fmt"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

// CalculateFetchPack determines which objects need to be fetched.
// Since the client cannot know the remote graph structure without fetching,
// this function currently only identifies the missing tips (refs).
// The actual dependency resolution happens during the fetch process.
func CalculateFetchPack(local []core.Hash, remote []core.Hash) []core.Hash {
	// Simple set difference: remote - local
	// This assumes 'local' contains what we have?
	// Usually 'local' refs are just tips. We should check if we HAVE the object.
	// But keeping signature simple.

	// Better: filter 'remote' hashes that are not in 'local' list?
	// No, we should check against the store roughly, but the function signature doesn't take store.
	// We'll rely on the caller to provide 'local' as a list of everything we have?
	// Unlikely. 'local' usually means 'local refs'.

	// Strategy: Return all remote hashes that are NOT present in local refs set.
	// The caller will then try to fetch them. If they exist locally, great.

	localSet := make(map[core.Hash]bool)
	for _, h := range local {
		localSet[h] = true
	}

	var needed []core.Hash
	for _, h := range remote {
		if !localSet[h] {
			needed = append(needed, h)
		}
	}
	return needed
}

// CalculatePushPack determines which objects need to be pushed.
// It traverses the graph from 'local' tips down, stopping at 'remote' tips.
func CalculatePushPack(store *storage.Store, local []core.Hash, remote []core.Hash) ([]core.Hash, error) {
	haveSet := make(map[core.Hash]bool)
	for _, h := range remote {
		haveSet[h] = true
	}

	visited := make(map[core.Hash]bool)
	var result []core.Hash

	// Queue for traversal
	queue := make([]core.Hash, len(local))
	copy(queue, local)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}

		// If remote has this object, we assume they have the history
		// Valid optimization for standard commits.
		if haveSet[current] {
			continue
		}

		// Get object to find children
		obj, err := store.Get(current)
		if err != nil {
			if err == core.ErrObjectNotFound {
				// We are missing a local object reachable from local ref? Corrupt repo?
				// Or maybe we just found a gap.
				return nil, fmt.Errorf("local object missing %s: %w", current, err)
			}
			return nil, err
		}

		// Add to result
		result = append(result, current)
		visited[current] = true

		// Add children to queue
		switch obj.Type {
		case core.ObjectTypeCommit:
			commit, err := core.DecodeCommit(obj.Data)
			if err != nil {
				return nil, err
			}
			queue = append(queue, commit.Tree)
			queue = append(queue, commit.Parents...)

		case core.ObjectTypeTree:
			tree, err := core.DecodeTree(obj.Data)
			if err != nil {
				return nil, err
			}
			for _, entry := range tree.Entries {
				queue = append(queue, entry.Hash)
			}

		case core.ObjectTypeBlob:
			// No children
		}
	}

	return result, nil
}

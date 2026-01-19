package transfer

import (
	"fmt"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

type FetcherClient interface {
	FetchObject(hash core.Hash) (*core.Object, error)
}

// Fetch performs a smart fetch using graph walking on the client side
func Fetch(store *storage.Store, client FetcherClient, remoteTips []core.Hash) error {
	queue := make([]core.Hash, len(remoteTips))
	copy(queue, remoteTips)

	visited := make(map[core.Hash]bool)

	// We can prioritize fetching.
	// And simplify by just walking everything until we hit an object we have.

	// This is valid: if we have the object in store, we assume we have its history.

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}

		if store.Exists(current) {
			visited[current] = true
			continue
		}

		// Fetch
		obj, err := client.FetchObject(current)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", current, err)
		}

		// Save
		if _, err := store.Put(obj.Type, obj.Data); err != nil {
			return fmt.Errorf("failed to save %s: %w", current, err)
		}

		visited[current] = true

		// Queue children
		switch obj.Type {
		case core.ObjectTypeCommit:
			commit, err := core.DecodeCommit(obj.Data)
			if err != nil {
				return err
			}
			queue = append(queue, commit.Tree)
			queue = append(queue, commit.Parents...)

		case core.ObjectTypeTree:
			tree, err := core.DecodeTree(obj.Data)
			if err != nil {
				return err
			}
			for _, entry := range tree.Entries {
				queue = append(queue, entry.Hash)
			}

		case core.ObjectTypeBlob:
			// No children
		}
	}

	return nil
}

package merge

import (
	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

// FindLCA finds the lowest common ancestor of two commits
func FindLCA(store *storage.Store, commit1, commit2 core.Hash) (core.Hash, error) {
	// Build ancestor sets for both commits
	ancestors1 := make(map[core.Hash]bool)
	ancestors2 := make(map[core.Hash]bool)

	// BFS from commit1
	queue1 := []core.Hash{commit1}
	for len(queue1) > 0 {
		hash := queue1[0]
		queue1 = queue1[1:]

		if ancestors1[hash] {
			continue
		}
		ancestors1[hash] = true

		commit, err := store.GetCommit(hash)
		if err != nil {
			continue // Reached root
		}

		if !commit.Parent.IsZero() {
			queue1 = append(queue1, commit.Parent)
		}
	}

	// BFS from commit2, looking for first common ancestor
	queue2 := []core.Hash{commit2}
	for len(queue2) > 0 {
		hash := queue2[0]
		queue2 = queue2[1:]

		if ancestors2[hash] {
			continue
		}
		ancestors2[hash] = true

		// Check if this is a common ancestor
		if ancestors1[hash] {
			return hash, nil
		}

		commit, err := store.GetCommit(hash)
		if err != nil {
			continue // Reached root
		}

		if !commit.Parent.IsZero() {
			queue2 = append(queue2, commit.Parent)
		}
	}

	return core.Hash{}, core.ErrNoCommonAncestor
}

// IsAncestor checks if ancestor is an ancestor of commit
func IsAncestor(store *storage.Store, ancestor, commit core.Hash) (bool, error) {
	if ancestor == commit {
		return true, nil
	}

	visited := make(map[core.Hash]bool)
	queue := []core.Hash{commit}

	for len(queue) > 0 {
		hash := queue[0]
		queue = queue[1:]

		if visited[hash] {
			continue
		}
		visited[hash] = true

		if hash == ancestor {
			return true, nil
		}

		c, err := store.GetCommit(hash)
		if err != nil {
			continue
		}

		if !c.Parent.IsZero() {
			queue = append(queue, c.Parent)
		}
	}

	return false, nil
}

// CanFastForward checks if we can fast-forward from base to target
func CanFastForward(store *storage.Store, base, target core.Hash) (bool, error) {
	// Fast-forward is possible if base is an ancestor of target
	return IsAncestor(store, base, target)
}

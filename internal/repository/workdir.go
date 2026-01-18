package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codimo/astral/internal/core"
	"golang.org/x/sync/errgroup"
)

// Save creates a new commit with the specified files and message
func (r *Repository) Save(files []string, message string) (core.Hash, error) {
	if message == "" {
		return core.Hash{}, fmt.Errorf("commit message cannot be empty")
	}

	// If no files specified, save all tracked files
	if len(files) == 0 {
		var err error
		files, err = r.listAllFiles()
		if err != nil {
			return core.Hash{}, err
		}
	}

	// Build tree from files
	tree, err := r.buildTree(files)
	if err != nil {
		return core.Hash{}, err
	}

	// Store tree
	treeHash, err := r.store.PutTree(tree)
	if err != nil {
		return core.Hash{}, err
	}

	// Get parent commit (if exists)
	var parentHash core.Hash
	currentCommit, err := r.GetCurrentCommit()
	if err == nil {
		parentHash = currentCommit
	}

	// Create commit
	var parents []core.Hash
	if !parentHash.IsZero() {
		parents = []core.Hash{parentHash}
	}
	commit := &core.Commit{
		Tree:      treeHash,
		Parents:   parents,
		Author:    r.getAuthorName(),
		Email:     r.getAuthorEmail(),
		Timestamp: time.Now(),
		Message:   message,
	}

	// Store commit
	commitHash, err := r.store.PutCommit(commit)
	if err != nil {
		return core.Hash{}, err
	}

	// Update branch reference
	branch, err := r.GetCurrentBranch()
	if err != nil {
		return core.Hash{}, err
	}

	ref := filepath.Join(headsDir, branch)
	if err := r.SetRef(ref, commitHash); err != nil {
		return core.Hash{}, err
	}

	return commitHash, nil
}

// buildTree creates a tree object from the given files
func (r *Repository) buildTree(files []string) (*core.Tree, error) {
	tree := &core.Tree{
		Entries: make([]core.TreeEntry, 0, len(files)),
	}

	// Use goroutines for parallel file hashing
	type result struct {
		entry core.TreeEntry
		err   error
	}

	results := make(chan result, len(files))
	var g errgroup.Group

	for _, file := range files {
		file := file // capture loop variable
		g.Go(func() error {
			// Get absolute path
			absPath := filepath.Join(r.Root, file)

			// Read file
			data, err := os.ReadFile(absPath)
			if err != nil {
				results <- result{err: fmt.Errorf("failed to read %s: %w", file, err)}
				return nil
			}

			// Store blob
			hash, err := r.store.PutBlob(data)
			if err != nil {
				results <- result{err: fmt.Errorf("failed to store %s: %w", file, err)}
				return nil
			}

			// Get file mode
			info, err := os.Stat(absPath)
			if err != nil {
				results <- result{err: fmt.Errorf("failed to stat %s: %w", file, err)}
				return nil
			}

			mode := uint32(0100644) // regular file
			if info.Mode()&0111 != 0 {
				mode = 0100755 // executable
			}

			results <- result{
				entry: core.TreeEntry{
					Mode: mode,
					Name: file,
					Hash: hash,
				},
			}
			return nil
		})
	}

	// Wait for all goroutines
	go func() {
		g.Wait()
		close(results)
	}()

	// Collect results
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		tree.Entries = append(tree.Entries, res.entry)
	}

	return tree, nil
}

// listAllFiles returns all non-ignored files in the repository
func (r *Repository) listAllFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(r.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .asl directory
		if info.IsDir() && info.Name() == aslDir {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(r.Root, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// Undo reverts the last commit but keeps working directory changes
func (r *Repository) Undo() error {
	// Get current commit
	currentHash, err := r.GetCurrentCommit()
	if err != nil {
		return err
	}

	// Get commit object
	commit, err := r.store.GetCommit(currentHash)
	if err != nil {
		return err
	}

	// Update branch to parent
	branch, err := r.GetCurrentBranch()
	if err != nil {
		return err
	}

	ref := filepath.Join(headsDir, branch)
	// Move to first parent (or zero hash if no parents)
	var parentHash core.Hash
	if len(commit.Parents) > 0 {
		parentHash = commit.Parents[0]
	}
	return r.SetRef(ref, parentHash)
}

// Amend modifies the last commit
func (r *Repository) Amend(files []string, message string) (core.Hash, error) {
	// Get current commit
	currentHash, err := r.GetCurrentCommit()
	if err != nil {
		return core.Hash{}, err
	}

	// Get commit object
	oldCommit, err := r.store.GetCommit(currentHash)
	if err != nil {
		return core.Hash{}, err
	}

	// If no message provided, use old message
	if message == "" {
		message = oldCommit.Message
	}

	// Build new tree
	if len(files) == 0 {
		files, err = r.listAllFiles()
		if err != nil {
			return core.Hash{}, err
		}
	}

	tree, err := r.buildTree(files)
	if err != nil {
		return core.Hash{}, err
	}

	treeHash, err := r.store.PutTree(tree)
	if err != nil {
		return core.Hash{}, err
	}

	// Create new commit with same parents as old commit
	commit := &core.Commit{
		Tree:      treeHash,
		Parents:   oldCommit.Parents,
		Author:    r.getAuthorName(),
		Email:     r.getAuthorEmail(),
		Timestamp: time.Now(),
		Message:   message,
	}

	commitHash, err := r.store.PutCommit(commit)
	if err != nil {
		return core.Hash{}, err
	}

	// Update branch reference
	branch, err := r.GetCurrentBranch()
	if err != nil {
		return core.Hash{}, err
	}

	ref := filepath.Join(headsDir, branch)
	if err := r.SetRef(ref, commitHash); err != nil {
		return core.Hash{}, err
	}

	return commitHash, nil
}

// GetCommitHistory returns the commit history starting from a hash
func (r *Repository) GetCommitHistory(startHash core.Hash, limit int) ([]*core.Commit, []core.Hash, error) {
	commits := make([]*core.Commit, 0)
	hashes := make([]core.Hash, 0)

	hash := startHash
	count := 0

	for !hash.IsZero() && (limit == 0 || count < limit) {
		commit, err := r.store.GetCommit(hash)
		if err != nil {
			return nil, nil, err
		}

		commits = append(commits, commit)
		hashes = append(hashes, hash)

		// Follow first parent for history
		if len(commit.Parents) > 0 {
			hash = commit.Parents[0]
		} else {
			hash = core.Hash{}
		}
		count++
	}

	return commits, hashes, nil
}

// getAuthorName returns the author name from config or environment
func (r *Repository) getAuthorName() string {
	if name := os.Getenv("ASL_AUTHOR_NAME"); name != "" {
		return name
	}
	if name := os.Getenv("USER"); name != "" {
		return name
	}
	return "Unknown"
}

// getAuthorEmail returns the author email from config or environment
func (r *Repository) getAuthorEmail() string {
	if email := os.Getenv("ASL_AUTHOR_EMAIL"); email != "" {
		return email
	}
	if email := os.Getenv("EMAIL"); email != "" {
		return email
	}
	return "unknown@localhost"
}

// Checkout restores files from a commit to the working directory
func (r *Repository) Checkout(commitHash core.Hash) error {
	commit, err := r.store.GetCommit(commitHash)
	if err != nil {
		return err
	}

	tree, err := r.store.GetTree(commit.Tree)
	if err != nil {
		return err
	}

	// Restore all files from tree
	for _, entry := range tree.Entries {
		obj, err := r.store.Get(entry.Hash)
		if err != nil {
			return fmt.Errorf("failed to get blob %s: %w", entry.Name, err)
		}

		if obj.Type != core.ObjectTypeBlob {
			continue
		}

		// Write file
		filePath := filepath.Join(r.Root, entry.Name)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		mode := os.FileMode(entry.Mode & 0777)
		if err := os.WriteFile(filePath, obj.Data, mode); err != nil {
			return fmt.Errorf("failed to write %s: %w", entry.Name, err)
		}
	}

	return nil
}

// Diff computes the difference between two trees
func (r *Repository) Diff(oldHash, newHash core.Hash) (map[string]string, error) {
	diff := make(map[string]string)

	var oldTree, newTree *core.Tree

	if !oldHash.IsZero() {
		oldCommit, err := r.store.GetCommit(oldHash)
		if err != nil {
			return nil, err
		}
		oldTree, err = r.store.GetTree(oldCommit.Tree)
		if err != nil {
			return nil, err
		}
	}

	if !newHash.IsZero() {
		newCommit, err := r.store.GetCommit(newHash)
		if err != nil {
			return nil, err
		}
		newTree, err = r.store.GetTree(newCommit.Tree)
		if err != nil {
			return nil, err
		}
	}

	// Build maps for comparison
	oldFiles := make(map[string]core.Hash)
	newFiles := make(map[string]core.Hash)

	if oldTree != nil {
		for _, entry := range oldTree.Entries {
			oldFiles[entry.Name] = entry.Hash
		}
	}

	if newTree != nil {
		for _, entry := range newTree.Entries {
			newFiles[entry.Name] = entry.Hash
		}
	}

	// Find modified and deleted files
	for name, oldHash := range oldFiles {
		newHash, exists := newFiles[name]
		if !exists {
			diff[name] = "deleted"
		} else if oldHash != newHash {
			diff[name] = "modified"
		}
	}

	// Find added files
	for name := range newFiles {
		if _, exists := oldFiles[name]; !exists {
			diff[name] = "added"
		}
	}

	return diff, nil
}

// GetFileContent retrieves file content from a commit
func (r *Repository) GetFileContent(commitHash core.Hash, filename string) ([]byte, error) {
	commit, err := r.store.GetCommit(commitHash)
	if err != nil {
		return nil, err
	}

	tree, err := r.store.GetTree(commit.Tree)
	if err != nil {
		return nil, err
	}

	for _, entry := range tree.Entries {
		if entry.Name == filename {
			obj, err := r.store.Get(entry.Hash)
			if err != nil {
				return nil, err
			}
			return obj.Data, nil
		}
	}

	return nil, core.ErrFileNotFound
}

package repository

import (
	"fmt"
	"time"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/merge"
)

// MergeOptions specifies options for a merge operation
type MergeOptions struct {
	NoFF     bool   // Force merge commit even if fast-forward
	FFOnly   bool   // Only merge if fast-forward possible
	Strategy string // "recursive" (default), "ours", "theirs"
}

// MergeResult represents the result of a merge operation
type MergeResult struct {
	FastForward bool
	Conflicts   bool
	MergeCommit *core.Hash
	Message     string
	AutoMerged  []string
	Conflicted  []string
}

// Merge merges the specified branch into the current branch
func (r *Repository) Merge(branch string, opts MergeOptions) (*MergeResult, error) {
	// 1. Check if merge already in progress
	if merge.IsMergeInProgress(r.Root) {
		return nil, core.ErrMergeInProgress
	}

	// 2. Resolve branch name to commit hash
	theirRef := fmt.Sprintf("refs/heads/%s", branch)
	theirCommit, err := r.GetRef(theirRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve branch %s: %w", branch, err)
	}

	// 3. Get current HEAD commit
	ourCommit, err := r.GetCurrentCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %w", err)
	}

	// 4. Find merge base (LCA)
	baseCommit, err := merge.FindLCA(r.store, ourCommit, theirCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	// 5. Check if fast-forward possible
	canFF, err := merge.CanFastForward(r.store, ourCommit, theirCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to check fast-forward: %w", err)
	}

	// 6. If FFOnly flag and can't FF, error
	if opts.FFOnly && !canFF {
		return nil, fmt.Errorf("cannot fast-forward")
	}

	// 7. If can FF and not NoFF, do fast-forward
	if canFF && !opts.NoFF {
		return r.doFastForward(theirCommit, branch)
	}

	// 8. Otherwise, do three-way merge
	return r.doThreeWayMerge(baseCommit, ourCommit, theirCommit, branch, opts)
}

// doFastForward performs a fast-forward merge
func (r *Repository) doFastForward(target core.Hash, branch string) (*MergeResult, error) {
	// Update HEAD to target commit
	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return nil, err
	}

	ref := fmt.Sprintf("refs/heads/%s", currentBranch)
	if err := r.SetRef(ref, target); err != nil {
		return nil, err
	}

	// Update working directory
	if err := r.Checkout(target); err != nil {
		return nil, err
	}

	return &MergeResult{
		FastForward: true,
		Conflicts:   false,
		MergeCommit: &target,
		Message:     fmt.Sprintf("Fast-forward to %s", branch),
	}, nil
}

// doThreeWayMerge performs a three-way merge
func (r *Repository) doThreeWayMerge(base, ours, theirs core.Hash, theirBranch string, opts MergeOptions) (*MergeResult, error) {
	// Get trees for base, ours, theirs
	baseTree, err := r.getCommitTree(base)
	if err != nil {
		return nil, err
	}

	ourTree, err := r.getCommitTree(ours)
	if err != nil {
		return nil, err
	}

	theirTree, err := r.getCommitTree(theirs)
	if err != nil {
		return nil, err
	}

	// Build file maps
	baseFiles := buildFileMap(baseTree)
	ourFiles := buildFileMap(ourTree)
	theirFiles := buildFileMap(theirTree)

	// Find all affected files
	allFiles := make(map[string]bool)
	for name := range baseFiles {
		allFiles[name] = true
	}
	for name := range ourFiles {
		allFiles[name] = true
	}
	for name := range theirFiles {
		allFiles[name] = true
	}

	// Merge each file
	var conflicts []merge.ConflictInfo
	var autoMerged []string
	mergedFiles := make(map[string]core.Hash)

	for filename := range allFiles {
		baseHash, baseExists := baseFiles[filename]
		ourHash, ourExists := ourFiles[filename]
		theirHash, theirExists := theirFiles[filename]

		// Handle different cases
		if !baseExists && ourExists && !theirExists {
			// Only in ours
			mergedFiles[filename] = ourHash
			autoMerged = append(autoMerged, filename)
		} else if !baseExists && !ourExists && theirExists {
			// Only in theirs
			mergedFiles[filename] = theirHash
			autoMerged = append(autoMerged, filename)
		} else if !baseExists && ourExists && theirExists {
			// Added in both
			if ourHash == theirHash {
				// Same content
				mergedFiles[filename] = ourHash
				autoMerged = append(autoMerged, filename)
			} else {
				// Different content - conflict
				conflicts = append(conflicts, merge.ConflictInfo{
					Path:     filename,
					Type:     "add-add",
					Resolved: false,
				})
			}
		} else if baseExists && !ourExists && !theirExists {
			// Deleted in both - OK
			continue
		} else if baseExists && !ourExists && theirExists {
			// Delete-modify conflict
			if baseHash == theirHash {
				// They didn't change it, we deleted it
				continue
			} else {
				conflicts = append(conflicts, merge.ConflictInfo{
					Path:     filename,
					Type:     "delete-modify",
					Resolved: false,
				})
			}
		} else if baseExists && ourExists && !theirExists {
			// Modify-delete conflict
			if baseHash == ourHash {
				// We didn't change it, they deleted it
				continue
			} else {
				conflicts = append(conflicts, merge.ConflictInfo{
					Path:     filename,
					Type:     "modify-delete",
					Resolved: false,
				})
			}
		} else if baseExists && ourExists && theirExists {
			// All three exist
			if ourHash == theirHash {
				// Both made same changes
				mergedFiles[filename] = ourHash
				autoMerged = append(autoMerged, filename)
			} else if baseHash == ourHash {
				// Only they changed it
				mergedFiles[filename] = theirHash
				autoMerged = append(autoMerged, filename)
			} else if baseHash == theirHash {
				// Only we changed it
				mergedFiles[filename] = ourHash
				autoMerged = append(autoMerged, filename)
			} else {
				// Both changed it differently - need content merge
				result, err := r.mergeFileContent(filename, baseHash, ourHash, theirHash)
				if err != nil {
					return nil, err
				}

				if result.HasConflict {
					conflicts = append(conflicts, merge.ConflictInfo{
						Path:     filename,
						Type:     "content",
						Resolved: false,
					})
				} else {
					// Store merged content
					hash, err := r.store.PutBlob([]byte(result.Content))
					if err != nil {
						return nil, err
					}
					mergedFiles[filename] = hash
					autoMerged = append(autoMerged, filename)
				}
			}
		}
	}

	// If conflicts exist, save merge state and return
	if len(conflicts) > 0 {
		currentBranch, err := r.GetCurrentBranch()
		if err != nil {
			return nil, err
		}

		state := &merge.MergeState{
			Branch:      theirBranch,
			BaseCommit:  base.String(),
			OurCommit:   ours.String(),
			TheirCommit: theirs.String(),
			Strategy:    opts.Strategy,
			Conflicts:   conflicts,
			Resolved:    []string{},
			AutoMerged:  autoMerged,
		}

		if err := merge.SaveMergeState(r.Root, state); err != nil {
			return nil, err
		}

		// Write conflict markers to files
		if err := r.writeConflictMarkers(conflicts, base, ours, theirs, currentBranch, theirBranch); err != nil {
			return nil, err
		}

		conflictPaths := make([]string, len(conflicts))
		for i, c := range conflicts {
			conflictPaths[i] = c.Path
		}

		return &MergeResult{
			FastForward: false,
			Conflicts:   true,
			Message:     fmt.Sprintf("Merge has conflicts in %d file(s)", len(conflicts)),
			AutoMerged:  autoMerged,
			Conflicted:  conflictPaths,
		}, nil
	}

	// No conflicts - create merge commit
	mergeCommit, err := r.createMergeCommit(theirBranch, ours, theirs, mergedFiles)
	if err != nil {
		return nil, err
	}

	// Update working directory
	if err := r.Checkout(mergeCommit); err != nil {
		return nil, err
	}

	return &MergeResult{
		FastForward: false,
		Conflicts:   false,
		MergeCommit: &mergeCommit,
		Message:     fmt.Sprintf("Merged %s into current branch", theirBranch),
		AutoMerged:  autoMerged,
	}, nil
}

// mergeFileContent performs three-way merge on file content
func (r *Repository) mergeFileContent(filename string, baseHash, ourHash, theirHash core.Hash) (*merge.MergeResult, error) {
	// Get file contents
	baseObj, err := r.store.Get(baseHash)
	if err != nil {
		return nil, err
	}

	ourObj, err := r.store.Get(ourHash)
	if err != nil {
		return nil, err
	}

	theirObj, err := r.store.Get(theirHash)
	if err != nil {
		return nil, err
	}

	// Perform three-way merge
	return merge.ThreeWayMerge(
		string(baseObj.Data),
		string(ourObj.Data),
		string(theirObj.Data),
		filename,
	), nil
}

// createMergeCommit creates a merge commit with two parents
func (r *Repository) createMergeCommit(theirBranch string, ourCommit, theirCommit core.Hash, files map[string]core.Hash) (core.Hash, error) {
	// Build tree from merged files
	tree := &core.Tree{
		Entries: make([]core.TreeEntry, 0, len(files)),
	}

	for filename, hash := range files {
		tree.Entries = append(tree.Entries, core.TreeEntry{
			Mode: 0100644,
			Name: filename,
			Hash: hash,
		})
	}

	// Store tree
	treeHash, err := r.store.PutTree(tree)
	if err != nil {
		return core.Hash{}, err
	}

	// Create commit with two parents
	commit := &core.Commit{
		Tree:      treeHash,
		Parents:   []core.Hash{ourCommit, theirCommit},
		Author:    r.getAuthorName(),
		Email:     r.getAuthorEmail(),
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Merge branch '%s'", theirBranch),
	}

	commitHash, err := r.store.PutCommit(commit)
	if err != nil {
		return core.Hash{}, err
	}

	// Update branch reference
	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return core.Hash{}, err
	}

	ref := fmt.Sprintf("refs/heads/%s", currentBranch)
	if err := r.SetRef(ref, commitHash); err != nil {
		return core.Hash{}, err
	}

	return commitHash, nil
}

// writeConflictMarkers writes conflict markers to files
func (r *Repository) writeConflictMarkers(conflicts []merge.ConflictInfo, base, ours, theirs core.Hash, ourBranch, theirBranch string) error {
	// For now, just write a simple conflict marker
	// TODO: Implement proper conflict marker generation
	return nil
}

// getCommitTree gets the tree for a commit
func (r *Repository) getCommitTree(commitHash core.Hash) (*core.Tree, error) {
	commit, err := r.store.GetCommit(commitHash)
	if err != nil {
		return nil, err
	}
	return r.store.GetTree(commit.Tree)
}

// buildFileMap builds a map of filename -> hash from a tree
func buildFileMap(tree *core.Tree) map[string]core.Hash {
	files := make(map[string]core.Hash)
	for _, entry := range tree.Entries {
		files[entry.Name] = entry.Hash
	}
	return files
}

// AbortMerge cancels an ongoing merge
func (r *Repository) AbortMerge() error {
	// 1. Load merge state
	state, err := merge.LoadMergeState(r.Root)
	if err != nil {
		return err
	}

	// 2. Restore working directory to pre-merge state (our commit)
	ourCommit, err := core.ParseHash(state.OurCommit)
	if err != nil {
		return err
	}

	if err := r.Checkout(ourCommit); err != nil {
		return err
	}

	// 3. Clear merge state
	return merge.ClearMergeState(r.Root)
}

// ContinueMerge completes a merge after conflict resolution
func (r *Repository) ContinueMerge() error {
	// 1. Load merge state
	state, err := merge.LoadMergeState(r.Root)
	if err != nil {
		return err
	}

	// 2. Validate all conflicts resolved
	if err := state.ValidateResolved(); err != nil {
		return err
	}

	// 3. Get all current files (including resolved conflicts)
	files, err := r.listAllFiles()
	if err != nil {
		return err
	}

	// 4. Build tree from current working directory
	tree, err := r.buildTree(files)
	if err != nil {
		return err
	}

	treeHash, err := r.store.PutTree(tree)
	if err != nil {
		return err
	}

	// 5. Parse commit hashes
	ourCommit, err := core.ParseHash(state.OurCommit)
	if err != nil {
		return err
	}

	theirCommit, err := core.ParseHash(state.TheirCommit)
	if err != nil {
		return err
	}

	// 6. Create merge commit
	commit := &core.Commit{
		Tree:      treeHash,
		Parents:   []core.Hash{ourCommit, theirCommit},
		Author:    r.getAuthorName(),
		Email:     r.getAuthorEmail(),
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Merge branch '%s'", state.Branch),
	}

	commitHash, err := r.store.PutCommit(commit)
	if err != nil {
		return err
	}

	// 7. Update branch reference
	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return err
	}

	ref := fmt.Sprintf("refs/heads/%s", currentBranch)
	if err := r.SetRef(ref, commitHash); err != nil {
		return err
	}

	// 8. Clear merge state
	return merge.ClearMergeState(r.Root)
}

package merge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codimo/astral/internal/core"
)

// MergeState tracks an ongoing merge operation
type MergeState struct {
	Branch      string         `json:"branch"`
	BaseCommit  string         `json:"base_commit"`
	OurCommit   string         `json:"our_commit"`
	TheirCommit string         `json:"their_commit"`
	Strategy    string         `json:"strategy"`
	Conflicts   []ConflictInfo `json:"conflicts"`
	Resolved    []string       `json:"resolved"`
	AutoMerged  []string       `json:"auto_merged"`
}

// ConflictInfo represents information about a conflict
type ConflictInfo struct {
	Path     string `json:"path"`
	Type     string `json:"type"` // "content", "delete-modify", "binary"
	Resolved bool   `json:"resolved"`
}

// SaveMergeState saves state to .asl/MERGE_STATE
func SaveMergeState(repoPath string, state *MergeState) error {
	stateFile := filepath.Join(repoPath, ".asl", "MERGE_STATE")

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal merge state: %w", err)
	}

	// Write to temp file first for atomic write
	tempFile := stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp merge state: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, stateFile); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to save merge state: %w", err)
	}

	return nil
}

// LoadMergeState loads from .asl/MERGE_STATE
func LoadMergeState(repoPath string) (*MergeState, error) {
	stateFile := filepath.Join(repoPath, ".asl", "MERGE_STATE")

	// Read file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, core.ErrNoMergeInProgress
		}
		return nil, fmt.Errorf("failed to read merge state: %w", err)
	}

	// Parse JSON
	var state MergeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse merge state: %w", err)
	}

	return &state, nil
}

// ClearMergeState removes .asl/MERGE_STATE
func ClearMergeState(repoPath string) error {
	stateFile := filepath.Join(repoPath, ".asl", "MERGE_STATE")

	err := os.Remove(stateFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear merge state: %w", err)
	}

	return nil
}

// IsMergeInProgress checks if MERGE_STATE exists
func IsMergeInProgress(repoPath string) bool {
	stateFile := filepath.Join(repoPath, ".asl", "MERGE_STATE")
	_, err := os.Stat(stateFile)
	return err == nil
}

// ValidateResolved ensures all conflicts are resolved
func (s *MergeState) ValidateResolved() error {
	for _, conflict := range s.Conflicts {
		if !conflict.Resolved {
			return core.ErrConflictsExist
		}
	}
	return nil
}

// MarkResolved marks a file as resolved
func (s *MergeState) MarkResolved(path string) error {
	found := false
	for i := range s.Conflicts {
		if s.Conflicts[i].Path == path {
			s.Conflicts[i].Resolved = true
			found = true
		}
	}

	if !found {
		return fmt.Errorf("no conflict found for file: %s", path)
	}

	// Add to resolved list if not already there
	for _, r := range s.Resolved {
		if r == path {
			return nil
		}
	}
	s.Resolved = append(s.Resolved, path)

	return nil
}

// HasUnresolvedConflicts checks if there are any unresolved conflicts
func (s *MergeState) HasUnresolvedConflicts() bool {
	for _, conflict := range s.Conflicts {
		if !conflict.Resolved {
			return true
		}
	}
	return false
}

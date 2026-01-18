package merge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/codimo/astral/internal/core"
)

func TestSaveAndLoadMergeState(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .asl directory
	aslDir := filepath.Join(tempDir, ".asl")
	if err := os.MkdirAll(aslDir, 0755); err != nil {
		t.Fatalf("Failed to create .asl dir: %v", err)
	}

	// Create test state
	state := &MergeState{
		Branch:      "feature-branch",
		BaseCommit:  "abc123",
		OurCommit:   "def456",
		TheirCommit: "ghi789",
		Strategy:    "recursive",
		Conflicts: []ConflictInfo{
			{Path: "file1.txt", Type: "content", Resolved: false},
			{Path: "file2.txt", Type: "binary", Resolved: false},
		},
		Resolved:   []string{},
		AutoMerged: []string{"file3.txt"},
	}

	// Save state
	if err := SaveMergeState(tempDir, state); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}

	// Load state
	loaded, err := LoadMergeState(tempDir)
	if err != nil {
		t.Fatalf("LoadMergeState failed: %v", err)
	}

	// Verify fields
	if loaded.Branch != state.Branch {
		t.Errorf("Branch mismatch: got %s, want %s", loaded.Branch, state.Branch)
	}
	if loaded.BaseCommit != state.BaseCommit {
		t.Errorf("BaseCommit mismatch: got %s, want %s", loaded.BaseCommit, state.BaseCommit)
	}
	if loaded.OurCommit != state.OurCommit {
		t.Errorf("OurCommit mismatch: got %s, want %s", loaded.OurCommit, state.OurCommit)
	}
	if loaded.TheirCommit != state.TheirCommit {
		t.Errorf("TheirCommit mismatch: got %s, want %s", loaded.TheirCommit, state.TheirCommit)
	}
	if loaded.Strategy != state.Strategy {
		t.Errorf("Strategy mismatch: got %s, want %s", loaded.Strategy, state.Strategy)
	}
	if len(loaded.Conflicts) != len(state.Conflicts) {
		t.Errorf("Conflicts length mismatch: got %d, want %d", len(loaded.Conflicts), len(state.Conflicts))
	}
	if len(loaded.AutoMerged) != len(state.AutoMerged) {
		t.Errorf("AutoMerged length mismatch: got %d, want %d", len(loaded.AutoMerged), len(state.AutoMerged))
	}
}

func TestMergeState_NoFile(t *testing.T) {
	// Create temp directory without state file
	tempDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .asl directory
	aslDir := filepath.Join(tempDir, ".asl")
	if err := os.MkdirAll(aslDir, 0755); err != nil {
		t.Fatalf("Failed to create .asl dir: %v", err)
	}

	// Try to load non-existent state
	_, err = LoadMergeState(tempDir)
	if err != core.ErrNoMergeInProgress {
		t.Errorf("Expected ErrNoMergeInProgress, got: %v", err)
	}
}

func TestMergeState_InvalidJSON(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .asl directory
	aslDir := filepath.Join(tempDir, ".asl")
	if err := os.MkdirAll(aslDir, 0755); err != nil {
		t.Fatalf("Failed to create .asl dir: %v", err)
	}

	// Write invalid JSON
	stateFile := filepath.Join(aslDir, "MERGE_STATE")
	if err := os.WriteFile(stateFile, []byte("invalid json {"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Try to load invalid state
	_, err = LoadMergeState(tempDir)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestMergeState_AtomicWrite(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .asl directory
	aslDir := filepath.Join(tempDir, ".asl")
	if err := os.MkdirAll(aslDir, 0755); err != nil {
		t.Fatalf("Failed to create .asl dir: %v", err)
	}

	// Create initial state
	state1 := &MergeState{
		Branch:      "branch1",
		BaseCommit:  "abc123",
		OurCommit:   "def456",
		TheirCommit: "ghi789",
		Strategy:    "recursive",
	}

	// Save initial state
	if err := SaveMergeState(tempDir, state1); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}

	// Create updated state
	state2 := &MergeState{
		Branch:      "branch2",
		BaseCommit:  "xyz999",
		OurCommit:   "uvw888",
		TheirCommit: "rst777",
		Strategy:    "ours",
	}

	// Save updated state (should overwrite)
	if err := SaveMergeState(tempDir, state2); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}

	// Load and verify we got the updated state
	loaded, err := LoadMergeState(tempDir)
	if err != nil {
		t.Fatalf("LoadMergeState failed: %v", err)
	}

	if loaded.Branch != state2.Branch {
		t.Errorf("Expected branch %s, got %s (atomic write may have failed)", state2.Branch, loaded.Branch)
	}

	// Verify temp file was cleaned up
	tempFile := filepath.Join(aslDir, "MERGE_STATE.tmp")
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temporary file was not cleaned up")
	}
}

func TestIsMergeInProgress(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .asl directory
	aslDir := filepath.Join(tempDir, ".asl")
	if err := os.MkdirAll(aslDir, 0755); err != nil {
		t.Fatalf("Failed to create .asl dir: %v", err)
	}

	// Should be false initially
	if IsMergeInProgress(tempDir) {
		t.Error("Expected no merge in progress")
	}

	// Save state
	state := &MergeState{
		Branch:      "test-branch",
		BaseCommit:  "abc123",
		OurCommit:   "def456",
		TheirCommit: "ghi789",
		Strategy:    "recursive",
	}
	if err := SaveMergeState(tempDir, state); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}

	// Should be true now
	if !IsMergeInProgress(tempDir) {
		t.Error("Expected merge in progress")
	}

	// Clear state
	if err := ClearMergeState(tempDir); err != nil {
		t.Fatalf("ClearMergeState failed: %v", err)
	}

	// Should be false again
	if IsMergeInProgress(tempDir) {
		t.Error("Expected no merge in progress after clear")
	}
}

func TestValidateResolved(t *testing.T) {
	tests := []struct {
		name      string
		state     *MergeState
		wantError bool
	}{
		{
			name: "no conflicts",
			state: &MergeState{
				Conflicts: []ConflictInfo{},
			},
			wantError: false,
		},
		{
			name: "all resolved",
			state: &MergeState{
				Conflicts: []ConflictInfo{
					{Path: "file1.txt", Type: "content", Resolved: true},
					{Path: "file2.txt", Type: "content", Resolved: true},
				},
			},
			wantError: false,
		},
		{
			name: "some unresolved",
			state: &MergeState{
				Conflicts: []ConflictInfo{
					{Path: "file1.txt", Type: "content", Resolved: true},
					{Path: "file2.txt", Type: "content", Resolved: false},
				},
			},
			wantError: true,
		},
		{
			name: "all unresolved",
			state: &MergeState{
				Conflicts: []ConflictInfo{
					{Path: "file1.txt", Type: "content", Resolved: false},
					{Path: "file2.txt", Type: "content", Resolved: false},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.ValidateResolved()
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestMarkResolved(t *testing.T) {
	state := &MergeState{
		Conflicts: []ConflictInfo{
			{Path: "file1.txt", Type: "content", Resolved: false},
			{Path: "file2.txt", Type: "binary", Resolved: false},
		},
		Resolved: []string{},
	}

	// Mark first file as resolved
	if err := state.MarkResolved("file1.txt"); err != nil {
		t.Fatalf("MarkResolved failed: %v", err)
	}

	// Check that file1 is marked resolved
	if !state.Conflicts[0].Resolved {
		t.Error("file1.txt should be marked as resolved")
	}
	if state.Conflicts[1].Resolved {
		t.Error("file2.txt should not be marked as resolved")
	}

	// Check that file1 is in resolved list
	if len(state.Resolved) != 1 || state.Resolved[0] != "file1.txt" {
		t.Error("file1.txt should be in resolved list")
	}

	// Try to mark non-existent file
	if err := state.MarkResolved("nonexistent.txt"); err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Mark same file again (should be idempotent)
	if err := state.MarkResolved("file1.txt"); err != nil {
		t.Fatalf("MarkResolved failed on second call: %v", err)
	}

	// Should still have only one entry in resolved list
	if len(state.Resolved) != 1 {
		t.Errorf("Expected 1 entry in resolved list, got %d", len(state.Resolved))
	}
}

func TestHasUnresolvedConflicts(t *testing.T) {
	tests := []struct {
		name  string
		state *MergeState
		want  bool
	}{
		{
			name: "no conflicts",
			state: &MergeState{
				Conflicts: []ConflictInfo{},
			},
			want: false,
		},
		{
			name: "all resolved",
			state: &MergeState{
				Conflicts: []ConflictInfo{
					{Path: "file1.txt", Type: "content", Resolved: true},
					{Path: "file2.txt", Type: "content", Resolved: true},
				},
			},
			want: false,
		},
		{
			name: "some unresolved",
			state: &MergeState{
				Conflicts: []ConflictInfo{
					{Path: "file1.txt", Type: "content", Resolved: true},
					{Path: "file2.txt", Type: "content", Resolved: false},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.HasUnresolvedConflicts()
			if got != tt.want {
				t.Errorf("HasUnresolvedConflicts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClearMergeState_NoFile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "astral-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .asl directory
	aslDir := filepath.Join(tempDir, ".asl")
	if err := os.MkdirAll(aslDir, 0755); err != nil {
		t.Fatalf("Failed to create .asl dir: %v", err)
	}

	// Should not error when clearing non-existent state
	if err := ClearMergeState(tempDir); err != nil {
		t.Errorf("ClearMergeState should not error on non-existent file: %v", err)
	}
}

func TestMergeState_JSONFormat(t *testing.T) {
	state := &MergeState{
		Branch:      "feature",
		BaseCommit:  "base123",
		OurCommit:   "our456",
		TheirCommit: "their789",
		Strategy:    "recursive",
		Conflicts: []ConflictInfo{
			{Path: "test.txt", Type: "content", Resolved: false},
		},
		Resolved:   []string{"resolved.txt"},
		AutoMerged: []string{"auto.txt"},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	// Unmarshal back
	var loaded MergeState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	// Verify round-trip
	if loaded.Branch != state.Branch {
		t.Errorf("Branch mismatch after round-trip")
	}
	if len(loaded.Conflicts) != len(state.Conflicts) {
		t.Errorf("Conflicts length mismatch after round-trip")
	}
}

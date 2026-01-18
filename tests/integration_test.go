package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codimo/astral/internal/repository"
)

func TestIntegrationBasicWorkflow(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "astral-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Verify .asl directory exists
	aslPath := filepath.Join(tmpDir, ".asl")
	if _, err := os.Stat(aslPath); os.IsNotExist(err) {
		t.Error(".asl directory should exist")
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, Astral!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Save (commit) the file
	hash1, err := repo.Save(nil, "Initial commit")
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	if hash1.IsZero() {
		t.Error("commit hash should not be zero")
	}

	// Modify file
	newContent := []byte("Hello, Astral v2!")
	if err := os.WriteFile(testFile, newContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Save second commit
	hash2, err := repo.Save(nil, "Update test file")
	if err != nil {
		t.Fatalf("failed to save second commit: %v", err)
	}

	if hash1 == hash2 {
		t.Error("different commits should have different hashes")
	}

	// Get commit history
	commits, hashes, err := repo.GetCommitHistory(hash2, 0)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(commits))
	}

	if hashes[0] != hash2 {
		t.Error("first commit in history should be latest")
	}

	if hashes[1] != hash1 {
		t.Error("second commit in history should be first commit")
	}

	// Check parent relationship
	if len(commits[0].Parents) == 0 || commits[0].Parents[0] != hash1 {
		t.Error("second commit should have first commit as parent")
	}

	if len(commits[1].Parents) != 0 {
		t.Error("first commit should have no parents")
	}
}

func TestIntegrationBranching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = repo.Save(nil, "Initial commit")
	if err != nil {
		t.Fatal(err)
	}

	// Create branch
	if err := repo.CreateBranch("feature"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// List branches
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatal(err)
	}

	if len(branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(branches))
	}

	// Switch to new branch
	if err := repo.SwitchBranch("feature"); err != nil {
		t.Fatalf("failed to switch branch: %v", err)
	}

	// Verify current branch
	branch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	if branch != "feature" {
		t.Errorf("expected branch 'feature', got '%s'", branch)
	}

	// Switch back to main
	if err := repo.SwitchBranch("main"); err != nil {
		t.Fatal(err)
	}

	branch, err = repo.GetCurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	if branch != "main" {
		t.Errorf("expected branch 'main', got '%s'", branch)
	}
}

func TestIntegrationUndoAndAmend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create commits
	testFile := filepath.Join(tmpDir, "file.txt")

	os.WriteFile(testFile, []byte("v1"), 0644)
	hash1, _ := repo.Save(nil, "First commit")

	os.WriteFile(testFile, []byte("v2"), 0644)
	_, _ = repo.Save(nil, "Second commit")

	// Undo last commit
	if err := repo.Undo(); err != nil {
		t.Fatalf("failed to undo: %v", err)
	}

	// Current commit should be first commit
	current, err := repo.GetCurrentCommit()
	if err != nil {
		t.Fatal(err)
	}

	if current != hash1 {
		t.Error("undo should move HEAD to previous commit")
	}

	// Amend commit
	os.WriteFile(testFile, []byte("v1 amended"), 0644)
	hash3, err := repo.Amend(nil, "First commit (amended)")
	if err != nil {
		t.Fatalf("failed to amend: %v", err)
	}

	if hash3 == hash1 {
		t.Error("amended commit should have different hash")
	}

	// Verify amended commit
	commit, err := repo.Store().GetCommit(hash3)
	if err != nil {
		t.Fatal(err)
	}

	if commit.Message != "First commit (amended)" {
		t.Error("amended commit should have new message")
	}
}

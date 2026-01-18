package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codimo/astral/internal/merge"
	"github.com/codimo/astral/internal/repository"
)

// TestMerge_FastForward tests a simple fast-forward merge
func TestMerge_FastForward(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-merge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit on main
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	hash1, err := repo.Save(nil, "Initial commit")
	if err != nil {
		t.Fatal(err)
	}

	// Create branch from main
	if err := repo.CreateBranch("feature"); err != nil {
		t.Fatal(err)
	}

	// Switch to feature and make a commit
	if err := repo.SwitchBranch("feature"); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file1, []byte("feature change"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = repo.Save(nil, "Feature commit")
	if err != nil {
		t.Fatal(err)
	}

	// Switch back to main
	if err := repo.SwitchBranch("main"); err != nil {
		t.Fatal(err)
	}

	// Verify main is still at initial commit
	currentCommit, err := repo.GetCurrentCommit()
	if err != nil {
		t.Fatal(err)
	}
	if currentCommit != hash1 {
		t.Error("main should still be at initial commit")
	}

	// Merge feature into main (should be fast-forward)
	result, err := repo.Merge("feature", repository.MergeOptions{})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	// Verify it was a fast-forward
	if !result.FastForward {
		t.Error("expected fast-forward merge")
	}

	if result.Conflicts {
		t.Error("expected no conflicts")
	}

	// Verify file content
	content, err := os.ReadFile(file1)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "feature change" {
		t.Errorf("expected 'feature change', got '%s'", string(content))
	}
}

// TestMerge_NoConflicts tests a three-way merge without conflicts
func TestMerge_NoConflicts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-merge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit with two files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("base1"), 0644)
	os.WriteFile(file2, []byte("base2"), 0644)
	_, err = repo.Save(nil, "Initial commit")
	if err != nil {
		t.Fatal(err)
	}

	// Create feature branch and modify file1
	repo.CreateBranch("feature")
	repo.SwitchBranch("feature")
	os.WriteFile(file1, []byte("feature1"), 0644)
	repo.Save(nil, "Feature commit")

	// Switch to main and modify file2 (different file)
	repo.SwitchBranch("main")
	os.WriteFile(file2, []byte("main2"), 0644)
	repo.Save(nil, "Main commit")

	// Merge feature into main (three-way merge, no conflicts)
	result, err := repo.Merge("feature", repository.MergeOptions{})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	// Verify it was NOT a fast-forward
	if result.FastForward {
		t.Error("expected three-way merge, got fast-forward")
	}

	// Verify no conflicts
	if result.Conflicts {
		t.Error("expected no conflicts")
	}

	// Verify both changes are present
	content1, _ := os.ReadFile(file1)
	content2, _ := os.ReadFile(file2)

	if string(content1) != "feature1" {
		t.Errorf("file1: expected 'feature1', got '%s'", string(content1))
	}
	if string(content2) != "main2" {
		t.Errorf("file2: expected 'main2', got '%s'", string(content2))
	}

	// Verify merge commit has two parents
	currentCommit, err := repo.GetCurrentCommit()
	if err != nil {
		t.Fatal(err)
	}

	commit, err := repo.Store().GetCommit(currentCommit)
	if err != nil {
		t.Fatal(err)
	}

	if len(commit.Parents) != 2 {
		t.Errorf("expected 2 parents, got %d", len(commit.Parents))
	}
}

// TestMerge_WithConflicts tests merge with conflicts
func TestMerge_WithConflicts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-merge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(file1, []byte("base content"), 0644)
	_, err = repo.Save(nil, "Initial commit")
	if err != nil {
		t.Fatal(err)
	}

	// Create feature branch and modify file
	repo.CreateBranch("feature")
	repo.SwitchBranch("feature")
	os.WriteFile(file1, []byte("feature content"), 0644)
	repo.Save(nil, "Feature commit")

	// Switch to main and modify same file differently
	repo.SwitchBranch("main")
	os.WriteFile(file1, []byte("main content"), 0644)
	repo.Save(nil, "Main commit")

	// Merge feature into main (should conflict)
	result, err := repo.Merge("feature", repository.MergeOptions{})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	// Verify conflict was detected
	if !result.Conflicts {
		t.Error("expected conflicts")
	}

	// Verify merge state was saved
	if !merge.IsMergeInProgress(tmpDir) {
		t.Error("expected merge to be in progress")
	}
}

// TestMerge_Abort tests aborting a merge
func TestMerge_Abort(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "astral-merge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repository.Init(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(file1, []byte("base"), 0644)
	initialCommit, err := repo.Save(nil, "Initial")
	if err != nil {
		t.Fatal(err)
	}

	// Create conflicting branches
	repo.CreateBranch("feature")
	repo.SwitchBranch("feature")
	os.WriteFile(file1, []byte("feature"), 0644)
	repo.Save(nil, "Feature")

	repo.SwitchBranch("main")
	os.WriteFile(file1, []byte("main"), 0644)
	repo.Save(nil, "Main")

	// Start merge (will conflict)
	repo.Merge("feature", repository.MergeOptions{})

	// Abort the merge
	if err := repo.AbortMerge(); err != nil {
		t.Fatalf("abort failed: %v", err)
	}

	// Verify no merge in progress
	if merge.IsMergeInProgress(tmpDir) {
		t.Error("merge should not be in progress after abort")
	}

	// Verify HEAD is at the pre-merge commit
	currentCommit, err := repo.GetCurrentCommit()
	if err != nil {
		t.Fatal(err)
	}

	// Current commit should be the "Main" commit, not initial
	if currentCommit == initialCommit {
		t.Error("HEAD should be at main commit, not initial")
	}
}

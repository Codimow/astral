package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

const (
	aslDir    = ".asl"
	configDir = "config"
	refsDir   = "refs"
	headsDir  = "refs/heads"
)

// Repository represents an Astral repository
type Repository struct {
	Root  string
	store *storage.Store
}

// Init initializes a new repository in the given directory
func Init(path string) (*Repository, error) {
	aslPath := filepath.Join(path, aslDir)

	// Check if already a repository
	if _, err := os.Stat(aslPath); err == nil {
		return nil, core.ErrAlreadyRepository
	}

	// Create .asl directory structure
	dirs := []string{
		aslPath,
		filepath.Join(aslPath, "objects"),
		filepath.Join(aslPath, "refs", "heads"),
		filepath.Join(aslPath, "config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create HEAD reference pointing to main
	headPath := filepath.Join(aslPath, "HEAD")
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		return nil, fmt.Errorf("failed to create HEAD: %w", err)
	}

	// Create default config
	configPath := filepath.Join(aslPath, "config", "config")
	defaultConfig := []byte("[core]\n\trepositoryformatversion = 1\n")
	if err := os.WriteFile(configPath, defaultConfig, 0644); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return Open(path)
}

// Open opens an existing repository
func Open(path string) (*Repository, error) {
	aslPath := filepath.Join(path, aslDir)

	// Check if .asl directory exists
	if _, err := os.Stat(aslPath); os.IsNotExist(err) {
		return nil, core.ErrNotARepository
	}

	return &Repository{
		Root:  path,
		store: storage.NewStore(aslPath),
	}, nil
}

// FindRoot finds the repository root by walking up the directory tree
func FindRoot(startPath string) (string, error) {
	path, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	for {
		aslPath := filepath.Join(path, aslDir)
		if _, err := os.Stat(aslPath); err == nil {
			return path, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", core.ErrNotARepository
		}
		path = parent
	}
}

// Store returns the object store
func (r *Repository) Store() *storage.Store {
	return r.store
}

// AslPath returns the .asl directory path
func (r *Repository) AslPath() string {
	return filepath.Join(r.Root, aslDir)
}

// GetHEAD returns the current HEAD reference
func (r *Repository) GetHEAD() (string, error) {
	headPath := filepath.Join(r.AslPath(), "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}

	// Parse "ref: refs/heads/main" format
	content := string(data)
	if len(content) > 5 && content[:5] == "ref: " {
		return content[5 : len(content)-1], nil
	}

	// Direct hash reference
	return content[:len(content)-1], nil
}

// SetHEAD sets the HEAD reference
func (r *Repository) SetHEAD(ref string) error {
	headPath := filepath.Join(r.AslPath(), "HEAD")

	var content string
	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		content = fmt.Sprintf("ref: %s\n", ref)
	} else {
		content = fmt.Sprintf("%s\n", ref)
	}

	return os.WriteFile(headPath, []byte(content), 0644)
}

// GetRef returns the hash that a reference points to
func (r *Repository) GetRef(ref string) (core.Hash, error) {
	refPath := filepath.Join(r.AslPath(), ref)
	data, err := os.ReadFile(refPath)
	if err != nil {
		if os.IsNotExist(err) {
			return core.Hash{}, core.ErrBranchNotFound
		}
		return core.Hash{}, fmt.Errorf("failed to read ref: %w", err)
	}

	hashStr := string(data)
	if len(hashStr) > 0 && hashStr[len(hashStr)-1] == '\n' {
		hashStr = hashStr[:len(hashStr)-1]
	}

	return core.ParseHash(hashStr)
}

// SetRef sets a reference to point to a hash
func (r *Repository) SetRef(ref string, hash core.Hash) error {
	refPath := filepath.Join(r.AslPath(), ref)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}

	content := fmt.Sprintf("%s\n", hash.String())
	return os.WriteFile(refPath, []byte(content), 0644)
}

// GetCurrentBranch returns the name of the current branch
func (r *Repository) GetCurrentBranch() (string, error) {
	ref, err := r.GetHEAD()
	if err != nil {
		return "", err
	}

	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		return ref[11:], nil
	}

	return "", fmt.Errorf("HEAD is detached")
}

// GetCurrentCommit returns the hash of the current commit
func (r *Repository) GetCurrentCommit() (core.Hash, error) {
	ref, err := r.GetHEAD()
	if err != nil {
		return core.Hash{}, err
	}

	// If HEAD points to a branch, resolve it
	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		return r.GetRef(ref)
	}

	// HEAD contains a direct hash
	return core.ParseHash(ref)
}

// ListBranches returns all branch names
func (r *Repository) ListBranches() ([]string, error) {
	headsPath := filepath.Join(r.AslPath(), headsDir)

	entries, err := os.ReadDir(headsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read branches: %w", err)
	}

	branches := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			branches = append(branches, entry.Name())
		}
	}

	return branches, nil
}

// CreateBranch creates a new branch pointing to the current commit
func (r *Repository) CreateBranch(name string) error {
	// Validate branch name
	if name == "" || name == "HEAD" {
		return core.ErrInvalidBranchName
	}

	// Check if branch already exists
	ref := filepath.Join(headsDir, name)
	refPath := filepath.Join(r.AslPath(), ref)
	if _, err := os.Stat(refPath); err == nil {
		return core.ErrBranchExists
	}

	// Get current commit
	currentCommit, err := r.GetCurrentCommit()
	if err != nil {
		if err == core.ErrBranchNotFound {
			// No commits yet, create empty branch
			return r.SetRef(ref, core.Hash{})
		}
		return err
	}

	return r.SetRef(ref, currentCommit)
}

// SwitchBranch switches to a different branch
func (r *Repository) SwitchBranch(name string) error {
	ref := filepath.Join(headsDir, name)
	refPath := filepath.Join(r.AslPath(), ref)

	// Check if branch exists
	if _, err := os.Stat(refPath); os.IsNotExist(err) {
		return core.ErrBranchNotFound
	}

	return r.SetHEAD(ref)
}

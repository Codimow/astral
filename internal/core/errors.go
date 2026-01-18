package core

import "errors"

var (
	// Repository errors
	ErrNotARepository    = errors.New("not an astral repository")
	ErrAlreadyRepository = errors.New("already an astral repository")
	ErrInvalidConfig     = errors.New("invalid configuration")

	// Object errors
	ErrObjectNotFound = errors.New("object not found")
	ErrInvalidObject  = errors.New("invalid object format")
	ErrInvalidHash    = errors.New("invalid hash")

	// Branch errors
	ErrBranchNotFound    = errors.New("branch not found")
	ErrBranchExists      = errors.New("branch already exists")
	ErrInvalidBranchName = errors.New("invalid branch name")

	// Commit errors
	ErrNoCommits       = errors.New("no commits yet")
	ErrInvalidCommit   = errors.New("invalid commit")
	ErrNothingToCommit = errors.New("nothing to commit")

	// Working directory errors
	ErrDirtyWorkingDir = errors.New("working directory has uncommitted changes")
	ErrFileNotFound    = errors.New("file not found")
)

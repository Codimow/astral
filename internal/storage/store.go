package storage

import (
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/codimo/astral/internal/core"
)

// Store manages the object database
type Store struct {
	root  string
	mu    sync.RWMutex
	cache map[core.Hash]*core.Object
}

// NewStore creates a new object store
func NewStore(root string) *Store {
	return &Store{
		root:  root,
		cache: make(map[core.Hash]*core.Object),
	}
}

// Put stores an object in the database
func (s *Store) Put(objType core.ObjectType, data []byte) (core.Hash, error) {
	// Create object with type prefix
	var obj []byte
	obj = append(obj, []byte(string(objType)+" ")...)
	obj = append(obj, data...)

	// Compute hash
	hash := core.HashBytes(obj)

	// Check if already exists
	path := s.objectPath(hash)
	if _, err := os.Stat(path); err == nil {
		return hash, nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return core.Hash{}, fmt.Errorf("failed to create object directory: %w", err)
	}

	// Write compressed object
	file, err := os.Create(path)
	if err != nil {
		return core.Hash{}, fmt.Errorf("failed to create object file: %w", err)
	}
	defer file.Close()

	writer := zlib.NewWriter(file)
	defer writer.Close()

	if _, err := writer.Write(obj); err != nil {
		return core.Hash{}, fmt.Errorf("failed to write object: %w", err)
	}

	return hash, nil
}

// Get retrieves an object from the database
func (s *Store) Get(hash core.Hash) (*core.Object, error) {
	// Check cache first
	s.mu.RLock()
	if obj, ok := s.cache[hash]; ok {
		s.mu.RUnlock()
		return obj, nil
	}
	s.mu.RUnlock()

	// Read from disk
	path := s.objectPath(hash)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, core.ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to open object: %w", err)
	}
	defer file.Close()

	// Decompress
	reader, err := zlib.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress object: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	// Parse object type
	typeEnd := -1
	for i, b := range data {
		if b == ' ' {
			typeEnd = i
			break
		}
	}

	if typeEnd == -1 {
		return nil, core.ErrInvalidObject
	}

	obj := &core.Object{
		Type: core.ObjectType(data[:typeEnd]),
		Data: data[typeEnd+1:],
		Hash: hash,
	}

	// Cache the object
	s.mu.Lock()
	s.cache[hash] = obj
	s.mu.Unlock()

	return obj, nil
}

// Exists checks if an object exists in the database
func (s *Store) Exists(hash core.Hash) bool {
	s.mu.RLock()
	_, cached := s.cache[hash]
	s.mu.RUnlock()

	if cached {
		return true
	}

	_, err := os.Stat(s.objectPath(hash))
	return err == nil
}

// objectPath returns the file path for a given hash
func (s *Store) objectPath(hash core.Hash) string {
	hashStr := hash.String()
	return filepath.Join(s.root, "objects", hashStr[:2], hashStr[2:])
}

// PutBlob stores a blob object
func (s *Store) PutBlob(data []byte) (core.Hash, error) {
	return s.Put(core.ObjectTypeBlob, data)
}

// PutTree stores a tree object
func (s *Store) PutTree(tree *core.Tree) (core.Hash, error) {
	data := core.EncodeTree(tree)
	return s.Put(core.ObjectTypeTree, data)
}

// PutCommit stores a commit object
func (s *Store) PutCommit(commit *core.Commit) (core.Hash, error) {
	data := core.EncodeCommit(commit)
	return s.Put(core.ObjectTypeCommit, data)
}

// GetCommit retrieves and decodes a commit object
func (s *Store) GetCommit(hash core.Hash) (*core.Commit, error) {
	obj, err := s.Get(hash)
	if err != nil {
		return nil, err
	}

	if obj.Type != core.ObjectTypeCommit {
		return nil, fmt.Errorf("expected commit, got %s", obj.Type)
	}

	return core.DecodeCommit(obj.Data)
}

// GetTree retrieves and decodes a tree object
func (s *Store) GetTree(hash core.Hash) (*core.Tree, error) {
	obj, err := s.Get(hash)
	if err != nil {
		return nil, err
	}

	if obj.Type != core.ObjectTypeTree {
		return nil, fmt.Errorf("expected tree, got %s", obj.Type)
	}

	return core.DecodeTree(obj.Data)
}

package core

import (
	"bytes"
	"fmt"
	"time"
)

// ObjectType represents the type of object stored in the database
type ObjectType string

const (
	ObjectTypeBlob   ObjectType = "blob"
	ObjectTypeTree   ObjectType = "tree"
	ObjectTypeCommit ObjectType = "commit"
)

// Object represents a generic object in the database
type Object struct {
	Type ObjectType
	Data []byte
	Hash Hash
}

// Commit represents a commit object
type Commit struct {
	Tree      Hash
	Parent    Hash
	Author    string
	Email     string
	Timestamp time.Time
	Message   string
}

// TreeEntry represents an entry in a tree object
type TreeEntry struct {
	Mode uint32
	Name string
	Hash Hash
}

// Tree represents a tree object
type Tree struct {
	Entries []TreeEntry
}

// EncodeCommit serializes a commit into bytes
func EncodeCommit(c *Commit) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "tree %s\n", c.Tree.String())
	if !c.Parent.IsZero() {
		fmt.Fprintf(&buf, "parent %s\n", c.Parent.String())
	}
	fmt.Fprintf(&buf, "author %s <%s> %d\n", c.Author, c.Email, c.Timestamp.Unix())
	fmt.Fprintf(&buf, "\n%s\n", c.Message)

	return buf.Bytes()
}

// DecodeCommit deserializes a commit from bytes
func DecodeCommit(data []byte) (*Commit, error) {
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 4 {
		return nil, ErrInvalidCommit
	}

	commit := &Commit{}
	messageStart := -1

	for i, line := range lines {
		if len(line) == 0 {
			messageStart = i + 1
			break
		}

		parts := bytes.SplitN(line, []byte(" "), 2)
		if len(parts) != 2 {
			continue
		}

		key := string(parts[0])
		value := parts[1]

		switch key {
		case "tree":
			hash, err := ParseHash(string(value))
			if err != nil {
				return nil, fmt.Errorf("invalid tree hash: %w", err)
			}
			commit.Tree = hash

		case "parent":
			hash, err := ParseHash(string(value))
			if err != nil {
				return nil, fmt.Errorf("invalid parent hash: %w", err)
			}
			commit.Parent = hash

		case "author":
			// Parse: "Name <email> timestamp"
			parts := bytes.Split(value, []byte(" "))
			if len(parts) < 3 {
				return nil, fmt.Errorf("invalid author format")
			}

			// Find email boundaries
			emailStart := bytes.IndexByte(value, '<')
			emailEnd := bytes.IndexByte(value, '>')
			if emailStart == -1 || emailEnd == -1 {
				return nil, fmt.Errorf("invalid email format")
			}

			commit.Author = string(bytes.TrimSpace(value[:emailStart]))
			commit.Email = string(value[emailStart+1 : emailEnd])

			// Parse timestamp
			var timestamp int64
			fmt.Sscanf(string(value[emailEnd+2:]), "%d", &timestamp)
			commit.Timestamp = time.Unix(timestamp, 0)
		}
	}

	if messageStart > 0 && messageStart < len(lines) {
		commit.Message = string(bytes.TrimSpace(bytes.Join(lines[messageStart:], []byte("\n"))))
	}

	return commit, nil
}

// EncodeTree serializes a tree into bytes
func EncodeTree(t *Tree) []byte {
	var buf bytes.Buffer

	for _, entry := range t.Entries {
		fmt.Fprintf(&buf, "%o %s\x00", entry.Mode, entry.Name)
		buf.Write(entry.Hash[:])
	}

	return buf.Bytes()
}

// DecodeTree deserializes a tree from bytes
func DecodeTree(data []byte) (*Tree, error) {
	tree := &Tree{
		Entries: make([]TreeEntry, 0),
	}

	for len(data) > 0 {
		// Find null terminator after mode and name
		nullIdx := bytes.IndexByte(data, 0)
		if nullIdx == -1 || nullIdx+32 > len(data) {
			break
		}

		// Parse mode and name
		parts := bytes.SplitN(data[:nullIdx], []byte(" "), 2)
		if len(parts) != 2 {
			return nil, ErrInvalidObject
		}

		var mode uint32
		fmt.Sscanf(string(parts[0]), "%o", &mode)

		entry := TreeEntry{
			Mode: mode,
			Name: string(parts[1]),
		}

		// Read hash
		copy(entry.Hash[:], data[nullIdx+1:nullIdx+33])

		tree.Entries = append(tree.Entries, entry)
		data = data[nullIdx+33:]
	}

	return tree, nil
}

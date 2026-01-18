package protocol

import "github.com/codimo/astral/internal/core"

// Reference updates
type RefUpdate struct {
	Name string
	Old  core.Hash
	New  core.Hash
}

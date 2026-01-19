package tests

import (
	"testing"

	"github.com/codimo/astral/internal/auth"
	"github.com/codimo/astral/internal/protocol"
)

func TestNewClient(t *testing.T) {
	c := protocol.NewClient("http://example.com", &auth.NoneAuth{})
	if c == nil {
		t.Error("NewClient returned nil")
	}
}

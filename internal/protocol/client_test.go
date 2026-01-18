package protocol

import (
	"testing"

	"github.com/codimo/astral/internal/auth"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://example.com", &auth.NoneAuth{})
	if c == nil {
		t.Error("NewClient returned nil")
	}
}

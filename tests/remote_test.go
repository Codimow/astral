package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codimo/astral/internal/remote"
)

func createTestRepoDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "astral-remote-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create .asl/config hierarchy
	configDir := filepath.Join(tempDir, ".asl", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config")
	if err := os.WriteFile(configFile, []byte("[core]\n\trepositoryformatversion = 1\n"), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	return tempDir
}

func TestRemoteOperations(t *testing.T) {
	repoPath := createTestRepoDir(t)
	defer os.RemoveAll(repoPath)

	// Test AddRemote
	err := remote.AddRemote(repoPath, "origin", "https://github.com/example/repo.git")
	if err != nil {
		t.Fatalf("AddRemote failed: %v", err)
	}

	// Test ListRemotes
	remotes, err := remote.ListRemotes(repoPath)
	if err != nil {
		t.Fatalf("ListRemotes failed: %v", err)
	}

	if len(remotes) != 1 {
		t.Errorf("Expected 1 remote, got %d", len(remotes))
	}

	if remotes[0].Name != "origin" {
		t.Errorf("Expected remote name 'origin', got '%s'", remotes[0].Name)
	}

	if remotes[0].URL != "https://github.com/example/repo.git" {
		t.Errorf("Expected remote URL 'https://github.com/example/repo.git', got '%s'", remotes[0].URL)
	}

	// Test duplicate AddRemote
	err = remote.AddRemote(repoPath, "origin", "https://github.com/example/other.git")
	if err == nil {
		t.Error("Expected error when adding duplicate remote, got nil")
	}

	// Test GetRemote
	r, err := remote.GetRemote(repoPath, "origin")
	if err != nil {
		t.Fatalf("GetRemote failed: %v", err)
	}
	if r.URL != "https://github.com/example/repo.git" {
		t.Errorf("GetRemote returned wrong URL")
	}

	// Test RemoveRemote
	err = remote.RemoveRemote(repoPath, "origin")
	if err != nil {
		t.Fatalf("RemoveRemote failed: %v", err)
	}

	remotes, err = remote.ListRemotes(repoPath)
	if err != nil {
		t.Fatalf("ListRemotes failed after remove: %v", err)
	}

	if len(remotes) != 0 {
		t.Errorf("Expected 0 remotes after remove, got %d", len(remotes))
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		url      string
		expected remote.RemoteURL
		wantErr  bool
	}{
		{
			url: "https://github.com/user/repo.git",
			expected: remote.RemoteURL{
				Protocol: "https",
				Host:     "github.com",
				Path:     "/user/repo.git",
			},
			wantErr: false,
		},
		{
			url: "http://localhost:8080/repo",
			expected: remote.RemoteURL{
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Path:     "/repo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		got, err := remote.ParseURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if got.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol mismatch: got %s, want %s", got.Protocol, tt.expected.Protocol)
			}
			if got.Host != tt.expected.Host {
				t.Errorf("Host mismatch: got %s, want %s", got.Host, tt.expected.Host)
			}
			if got.Port != tt.expected.Port {
				t.Errorf("Port mismatch: got %d, want %d", got.Port, tt.expected.Port)
			}
			if got.Path != tt.expected.Path {
				t.Errorf("Path mismatch: got %s, want %s", got.Path, tt.expected.Path)
			}
		}
	}
}

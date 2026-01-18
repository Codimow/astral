package remote

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Remote struct {
	Name     string
	URL      string
	FetchURL string // Optional, defaults to URL
	PushURL  string // Optional, defaults to URL
}

type RemoteConfig struct {
	Remotes map[string]Remote
}

type RemoteURL struct {
	Protocol string // "https", "ssh", "file"
	Host     string
	Port     int
	Path     string
	User     string
}

// AddRemote adds a new remote to the configuration
func AddRemote(repoPath, name, remoteURL string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}
	if remoteURL == "" {
		return fmt.Errorf("remote URL cannot be empty")
	}

	configPath := filepath.Join(repoPath, ".asl", "config", "config")

	// Check if already exists in the config map (we'll read it first)
	remotes, err := ListRemotes(repoPath)
	if err == nil {
		for _, r := range remotes {
			if r.Name == name {
				return fmt.Errorf("remote '%s' already exists", name)
			}
		}
	} else {
		// If ListRemotes fails, it might be because the file doesn't exist or is empty,
		// but we should proceed with caution/creation if needed.
		// Ideally checking if the file exists is better.
	}

	// Append to file
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("\n[remote \"%s\"]\n\turl = %s\n\tfetch = +refs/heads/*:refs/remotes/%s/*\n", name, remoteURL, name)); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	return nil
}

// RemoveRemote removes a remote from the configuration
func RemoveRemote(repoPath, name string) error {
	configPath := filepath.Join(repoPath, ".asl", "config", "config")

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inSection := false
	sectionHeader := fmt.Sprintf("[remote \"%s\"]", name)

	found := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == sectionHeader {
			inSection = true
			found = true
			continue
		}

		if inSection && strings.HasPrefix(trimmed, "[") {
			inSection = false
		}

		if !inSection {
			newLines = append(newLines, line)
		}
	}

	if !found {
		return fmt.Errorf("remote '%s' not found", name)
	}

	// Reconstruct file
	output := strings.Join(newLines, "\n")
	// Clean up potential multiple newlines
	// (Simple implementation, might create gaps but functional)

	return os.WriteFile(configPath, []byte(output), 0644)
}

// ListRemotes returns all configured remotes
func ListRemotes(repoPath string) ([]Remote, error) {
	configPath := filepath.Join(repoPath, ".asl", "config", "config")

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	remotes := make(map[string]*Remote)
	scanner := bufio.NewScanner(file)

	var currentRemote *Remote

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[remote \"") && strings.HasSuffix(line, "\"]") {
			name := line[9 : len(line)-2]
			currentRemote = &Remote{Name: name}
			remotes[name] = currentRemote
			continue
		}

		if strings.HasPrefix(line, "[") {
			currentRemote = nil // Other sections
			continue
		}

		if currentRemote != nil {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "url":
					currentRemote.URL = value
					if currentRemote.FetchURL == "" {
						currentRemote.FetchURL = value
					}
					if currentRemote.PushURL == "" {
						currentRemote.PushURL = value
					}
				case "pushurl":
					currentRemote.PushURL = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var result []Remote
	for _, r := range remotes {
		result = append(result, *r)
	}

	return result, nil
}

// GetRemote returns a specific remote by name
func GetRemote(repoPath, name string) (*Remote, error) {
	remotes, err := ListRemotes(repoPath)
	if err != nil {
		return nil, err
	}

	for _, r := range remotes {
		if r.Name == name {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("remote '%s' not found", name)
}

// ParseURL parses a remote URL into its components
func ParseURL(rawURL string) (*RemoteURL, error) {
	// Custom parsing to handle SCP-like syntax (user@host:path) if we supported SSH,
	// but sticking to standard URL parsing for Phase 3 HTTP/HTTPS focus primarily,
	// though standard URL parser handles most schemes.

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	port := 0
	if u.Port() != "" {
		p, err := strconv.Atoi(u.Port())
		if err == nil {
			port = p
		}
	}

	return &RemoteURL{
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Port:     port,
		Path:     u.Path,
		User:     u.User.Username(),
	}, nil
}

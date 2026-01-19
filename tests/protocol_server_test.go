package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/codimo/astral/internal/auth"
	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/protocol"
	"github.com/codimo/astral/internal/repository"
)

func createTestRepo(t *testing.T) *repository.Repository {
	dir, err := os.MkdirTemp("", "protocol-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repo, err := repository.Init(dir)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("Failed to init repo: %v", err)
	}
	return repo
}

func TestServer_InfoRefs(t *testing.T) {
	repo := createTestRepo(t)
	defer os.RemoveAll(repo.Root)

	server := protocol.NewServer(repo.Store(), repo, &auth.NoneAuth{})
	ts := httptest.NewServer(server)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/info/refs")
	if err != nil {
		t.Fatalf("Failed to get info/refs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var refs map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&refs); err != nil {
		t.Fatalf("Failed to decode refs: %v", err)
	}
}

func TestServer_Objects(t *testing.T) {
	repo := createTestRepo(t)
	defer os.RemoveAll(repo.Root)

	server := protocol.NewServer(repo.Store(), repo, &auth.NoneAuth{})
	ts := httptest.NewServer(server)
	defer ts.Close()

	// 1. Test POST /objects/ (Put)
	objData := []byte("hello world")
	obj := &core.Object{
		Type: core.ObjectTypeBlob,
		Data: objData,
	}

	payload, err := json.Marshal([]*core.Object{obj})
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Note: trailing slash matches endpoint definition /objects/
	resp, err := http.Post(ts.URL+"/objects/", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to post object: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
	}

	expectedHash := core.HashBytes(append([]byte("blob "), objData...))

	if !repo.Store().Exists(expectedHash) {
		t.Error("Posted object not found in store")
	}

	// 2. Test GET /objects/{hash}
	resp, err = http.Get(ts.URL + "/objects/" + expectedHash.String())
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var fetchedObj core.Object
	if err := json.NewDecoder(resp.Body).Decode(&fetchedObj); err != nil {
		t.Fatalf("Failed to decode object: %v", err)
	}

	if string(fetchedObj.Data) != string(objData) {
		t.Errorf("Expected data %q, got %q", objData, fetchedObj.Data)
	}

	if fetchedObj.Type != core.ObjectTypeBlob {
		t.Errorf("Expected type blob, got %s", fetchedObj.Type)
	}
}

package tests

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/codimo/astral/internal/auth"
	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/protocol"
)

func TestClient_Integration(t *testing.T) {
	// Setup Server
	repo := createTestRepo(t)
	defer os.RemoveAll(repo.Root)

	server := protocol.NewServer(repo.Store(), repo, &auth.NoneAuth{})
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Setup Client
	client := protocol.NewClient(ts.URL, &auth.NoneAuth{})

	// 1. Test PushObject
	objData := []byte("client test")
	obj := &core.Object{
		Type: core.ObjectTypeBlob,
		Data: objData,
	}
	// Hash is ignored by server but let's be nice

	if err := client.PushObject(obj); err != nil {
		t.Fatalf("PushObject failed: %v", err)
	}

	expectedHash := core.HashBytes(append([]byte("blob "), objData...))

	// Verify on server repo
	if !repo.Store().Exists(expectedHash) {
		t.Fatal("Object not saved in server repo")
	}

	// 2. Test FetchObject
	fetched, err := client.FetchObject(expectedHash)
	if err != nil {
		t.Fatalf("FetchObject failed: %v", err)
	}

	if string(fetched.Data) != string(objData) {
		t.Error("Fetched data mismatch")
	}

	// 3. Test UpdateRef
	// Create update
	if err := client.UpdateRef("refs/heads/main", expectedHash); err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	// Verify on server
	serverRef, err := repo.GetRef("refs/heads/main")
	if err != nil {
		t.Fatal("Server ref not updated")
	}
	if serverRef != expectedHash {
		t.Error("Server ref mismatch")
	}

	// 4. Test GetRef
	clientRef, err := client.GetRef("refs/heads/main")
	if err != nil {
		t.Fatalf("GetRef failed: %v", err)
	}
	if clientRef != expectedHash {
		t.Error("Client ref mismatch")
	}

	// 5. Test ListRefs
	refs, err := client.ListRefs()
	if err != nil {
		t.Fatalf("ListRefs failed: %v", err)
	}

	if h, ok := refs["refs/heads/main"]; !ok || h != expectedHash {
		t.Error("ListRefs missing main path")
	}
}

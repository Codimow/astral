package protocol

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/codimo/astral/internal/auth"
	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/storage"
)

// RefStore abstraction to avoid circular dependency with repository package
type RefStore interface {
	GetHEAD() (string, error)
	GetCurrentCommit() (core.Hash, error)
	ListBranches() ([]string, error)
	GetRef(ref string) (core.Hash, error)
	SetRef(ref string, hash core.Hash) error
}

type Server struct {
	store *storage.Store
	refs  RefStore
	auth  auth.Authenticator
	mux   *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(store *storage.Store, refs RefStore, auth auth.Authenticator) *Server {
	s := &Server{
		store: store,
		refs:  refs,
		auth:  auth,
		mux:   http.NewServeMux(),
	}

	s.mux.HandleFunc("/info/refs", s.handleInfoRefs)
	s.mux.HandleFunc("/objects/", s.handleObjectRequest) // /objects/{hash} and POST /objects
	s.mux.HandleFunc("/refs/heads/", s.handleRefRequest) // GET/POST /refs/heads/{branch}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		if err := s.auth.Authenticate(r); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}

// handleInfoRefs lists available refs (GET /info/refs)
func (s *Server) handleInfoRefs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	refs := make(map[string]string)

	// Get HEAD
	_, err := s.refs.GetHEAD()
	if err == nil {
		hash, err := s.refs.GetCurrentCommit()
		if err == nil {
			refs["HEAD"] = hash.String()
		}
	}

	// List branches
	branches, err := s.refs.ListBranches()
	if err == nil {
		for _, b := range branches {
			hash, err := s.refs.GetRef("refs/heads/" + b)
			if err == nil {
				refs["refs/heads/"+b] = hash.String()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refs)
}

// handleObjectRequest handles /objects/{hash} (GET) and /objects (POST)
func (s *Server) handleObjectRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/objects")

	if r.Method == http.MethodGet {
		// GET /objects/{hash}
		hashStr := strings.TrimPrefix(path, "/")
		if hashStr == "" {
			http.Error(w, "Missing hash", http.StatusBadRequest)
			return
		}

		hash, err := core.ParseHash(hashStr)
		if err != nil {
			http.Error(w, "Invalid hash: "+err.Error(), http.StatusBadRequest)
			return
		}

		obj, err := s.store.Get(hash)
		if err != nil {
			if err == core.ErrObjectNotFound {
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
				http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj)
		return
	}

	if r.Method == http.MethodPost {
		// POST /objects (Batch upload)
		var objects []*core.Object

		// Handle gzip compression
		var reader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			reader = gz
		}

		if err := json.NewDecoder(reader).Decode(&objects); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		for _, obj := range objects {
			_, err := s.store.Put(obj.Type, obj.Data)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to store object: %v", err), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleRefRequest handles /refs/heads/{branch}
func (s *Server) handleRefRequest(w http.ResponseWriter, r *http.Request) {
	branch := strings.TrimPrefix(r.URL.Path, "/refs/heads/")
	if branch == "" {
		http.Error(w, "Missing branch name", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		hash, err := s.refs.GetRef("refs/heads/" + branch)
		if err != nil {
			http.Error(w, "Ref not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"hash": hash.String()})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Hash string `json:"hash"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		newHash, err := core.ParseHash(req.Hash)
		if err != nil {
			http.Error(w, "Invalid hash: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.refs.SetRef("refs/heads/"+branch, newHash); err != nil {
			http.Error(w, "Failed to update ref: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

package protocol

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/codimo/astral/internal/auth"
	"github.com/codimo/astral/internal/core"
)

type Client struct {
	baseURL string
	auth    auth.Authenticator
	client  *http.Client
}

// NewClient creates a new client
func NewClient(url string, auth auth.Authenticator) *Client {
	// Ensure baseURL ends with / to avoid issues or handle in join
	// Standardize to NOT end with / usually, but easy to join with /
	return &Client{
		baseURL: strings.TrimSuffix(url, "/"),
		auth:    auth,
		client:  &http.Client{},
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	if c.auth != nil {
		if err := c.auth.Authenticate(req); err != nil {
			return nil, err
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		// Could enable gzip encoding for upload if nice usage
	}

	return c.client.Do(req)
}

// ListRefs lists refs on remote
func (c *Client) ListRefs() (map[string]core.Hash, error) {
	resp, err := c.doRequest(http.MethodGet, "/info/refs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote error: %s", resp.Status)
	}

	var rawRefs map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&rawRefs); err != nil {
		return nil, err
	}

	refs := make(map[string]core.Hash)
	for name, hashStr := range rawRefs {
		hash, err := core.ParseHash(hashStr)
		if err != nil {
			// Warn but continue? Or fail? Fail for data integrity.
			return nil, fmt.Errorf("invalid hash for ref %s: %w", name, err)
		}
		refs[name] = hash
	}

	return refs, nil
}

// FetchObject fetch object from remote
func (c *Client) FetchObject(hash core.Hash) (*core.Object, error) {
	resp, err := c.doRequest(http.MethodGet, "/objects/"+hash.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, core.ErrObjectNotFound
		}
		return nil, fmt.Errorf("remote error: %s", resp.Status)
	}

	var obj core.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, err
	}

	return &obj, nil
}

// FetchObjects fetches multiple objects efficiently
// In simple HTTP protocol without batch fetch endpoint, we call sequentially.
// Phase 3 spec says "Fetch remote objects" but didn't explicitly mandate a batch GET,
// though `CalculateFetchPack` implies bulk optimization.
// Protocol.go design showed `GET /objects/{hash}`.
// Adding a `POST /objects/batch-fetch` would be optimized, but let's stick to simple parallel GETs or sequential for now.
// Wait, the spec Client interface has `FetchObjects`.
func (c *Client) FetchObjects(hashes []core.Hash) ([]*core.Object, error) {
	// Current server implementation only supports single object GET.
	// We can loop.
	var objects []*core.Object
	for _, h := range hashes {
		obj, err := c.FetchObject(h)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// PushObject push object to remote
func (c *Client) PushObject(obj *core.Object) error {
	return c.PushObjects([]*core.Object{obj})
}

// PushObjects pushes multiple objects to remote
func (c *Client) PushObjects(objs []*core.Object) error {
	// Use gzip for batch upload
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(gw).Encode(objs); err != nil {
		return err
	}
	gw.Close()

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/objects/", &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if c.auth != nil {
		if err := c.auth.Authenticate(req); err != nil {
			return err
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// UpdateRef updates remote ref
func (c *Client) UpdateRef(ref string, hash core.Hash) error {
	payload := map[string]string{
		"hash": hash.String(),
	}
	data, _ := json.Marshal(payload)

	resp, err := c.doRequest(http.MethodPost, "/refs/heads/"+strings.TrimPrefix(ref, "refs/heads/"), bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// GetRef get remote ref
func (c *Client) GetRef(ref string) (core.Hash, error) {
	resp, err := c.doRequest(http.MethodGet, "/refs/heads/"+strings.TrimPrefix(ref, "refs/heads/"), nil)
	if err != nil {
		return core.Hash{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return core.Hash{}, core.ErrBranchNotFound
		}
		return core.Hash{}, fmt.Errorf("remote error: %s", resp.Status)
	}

	var res map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return core.Hash{}, err
	}

	return core.ParseHash(res["hash"])
}

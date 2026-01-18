package auth

import "net/http"

type Authenticator interface {
	Authenticate(*http.Request) error
}

// No authentication
type NoneAuth struct{}

func (a *NoneAuth) Authenticate(r *http.Request) error {
	return nil
}

// HTTP Basic Auth
type BasicAuth struct {
	Username string
	Password string
}

func (a *BasicAuth) Authenticate(r *http.Request) error {
	r.SetBasicAuth(a.Username, a.Password)
	return nil
}

// Token-based auth
type TokenAuth struct {
	Token string
}

func (a *TokenAuth) Authenticate(r *http.Request) error {
	r.Header.Set("Authorization", "Bearer "+a.Token)
	return nil
}

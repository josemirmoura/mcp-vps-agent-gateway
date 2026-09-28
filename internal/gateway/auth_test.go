package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

func TestStaticAuthUsesNativeMCPMiddleware(t *testing.T) {
	var gotSubject string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSubject = SubjectFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	h := (AuthConfig{Mode: "static", StaticToken: "secret", StaticSubject: "alice"}).Wrap(next)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent || gotSubject != "alice" {
		t.Fatalf("status=%d subject=%q", w.Code, gotSubject)
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d", w.Code)
	}
}

func TestOAuthChallengeAdvertisesResourceMetadata(t *testing.T) {
	metadataURL := "https://mcp.example.com/.well-known/oauth-protected-resource"
	verifier := mcpauth.TokenVerifier(func(context.Context, string, *http.Request) (*mcpauth.TokenInfo, error) {
		return nil, mcpauth.ErrInvalidToken
	})
	h := (AuthConfig{
		Mode:                "integrated",
		Verifier:            verifier,
		ResourceMetadataURL: metadataURL,
	}).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusUnauthorized)
	}
	challenge := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(strings.ToLower(challenge), "bearer") {
		t.Fatalf("WWW-Authenticate missing Bearer challenge: %q", challenge)
	}
	if !strings.Contains(challenge, `resource_metadata="`+metadataURL+`"`) {
		t.Fatalf("WWW-Authenticate missing resource metadata URL: %q", challenge)
	}
}

func TestConfiguredVerifierSubject(t *testing.T) {
	verifier := mcpauth.TokenVerifier(func(context.Context, string, *http.Request) (*mcpauth.TokenInfo, error) {
		return &mcpauth.TokenInfo{UserID: "subject-123"}, nil
	})
	var got string
	h := (AuthConfig{Mode: "integrated", Verifier: verifier}).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = SubjectFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer whatever")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	// Native middleware requires expiration unless explicitly relaxed.
	if w.Code == http.StatusNoContent {
		t.Fatalf("expected missing expiration rejection; subject=%q", got)
	}
}

func TestProtectedResourceMetadata(t *testing.T) {
	h := AuthConfig{
		ResourceIdentifier:   "https://mcp.example.com/mcp",
		AuthorizationServers: []string{"https://auth.example.com"},
		RequiredScopes:       []string{"vps.read", "vps.write"},
	}.ProtectedResourceMetadataHandler()
	if h == nil {
		t.Fatal("expected protected resource metadata handler")
	}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`"resource":"https://mcp.example.com/mcp"`,
		`"authorization_servers":["https://auth.example.com"]`,
		`"bearer_methods_supported":["header"]`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata missing %s: %s", want, body)
		}
	}
}


func TestIntegratedOIDCVerifierUsesAudienceBoundPrivateIntrospection(t *testing.T) {
	audience := "resource-project-123"
	audienceScope := "urn:zitadel:iam:org:project:id:" + audience + ":aud"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/v2/introspect" {
			http.NotFound(w, r)
			return
		}
		if r.Host != "mcp.example.com" || r.Header.Get("X-Forwarded-Proto") != "https" {
			http.Error(w, "bad forwarded identity", http.StatusBadRequest)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "introspector" || pass != "introspection-secret" {
			http.Error(w, "bad client auth", http.StatusUnauthorized)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("token") != "good-token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"active":false}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"active":true,
			"sub":"operator-123",
			"scope":"openid ` + audienceScope + `",
			"exp":` + strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10) + `,
			"iss":"https://mcp.example.com",
			"aud":["` + audience + `","another-audience"]
		}`))
	}))
	defer server.Close()

	verifier, err := NewIntegratedOIDCVerifier(
		server.URL+"/oauth/v2/introspect",
		"mcp.example.com",
		"introspector",
		"introspection-secret",
		"https://mcp.example.com",
		audience,
	)
	if err != nil {
		t.Fatal(err)
	}

	info, err := verifier(context.Background(), "good-token", httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if err != nil {
		t.Fatal(err)
	}
	if info.UserID != "operator-123" {
		t.Fatalf("subject=%q", info.UserID)
	}
	if time.Until(info.Expiration) <= 0 {
		t.Fatalf("expected future expiration, got %v", info.Expiration)
	}
	if !slices.Contains(info.Scopes, audienceScope) {
		t.Fatalf("missing audience scope in %v", info.Scopes)
	}

	if _, err := verifier(context.Background(), "bad-token", httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
		t.Fatal("expected inactive token rejection")
	}
}

func TestIntegratedOIDCVerifierRejectsWrongAudienceOrIssuer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"active":true,
			"sub":"operator-123",
			"scope":"openid",
			"exp":` + strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10) + `,
			"iss":"https://mcp.example.com",
			"aud":["resource-project-123"]
		}`))
	}))
	defer server.Close()

	for _, tc := range []struct {
		name     string
		issuer   string
		audience string
	}{
		{name: "issuer", issuer: "https://other.example.com", audience: "resource-project-123"},
		{name: "audience", issuer: "https://mcp.example.com", audience: "resource-project-999"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifier, err := NewIntegratedOIDCVerifier(
				server.URL,
				"",
				"introspector",
				"introspection-secret",
				tc.issuer,
				tc.audience,
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := verifier(context.Background(), "token", httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
				t.Fatal("expected token rejection")
			}
		})
	}
}

func TestIntegratedOIDCVerifierRejectsExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"active":true,
			"sub":"operator-123",
			"scope":"openid",
			"exp":` + strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10) + `,
			"iss":"https://mcp.example.com",
			"aud":["resource-project-123"]
		}`))
	}))
	defer server.Close()

	verifier, err := NewIntegratedOIDCVerifier(
		server.URL,
		"",
		"introspector",
		"introspection-secret",
		"https://mcp.example.com",
		"resource-project-123",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier(context.Background(), "expired-token", httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
		t.Fatal("expected expired token rejection")
	}
}

func TestIntegratedOIDCVerifierRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name, endpoint, clientID, clientSecret, issuer, audience string
	}{
		{name: "endpoint empty", clientID: "id", clientSecret: "secret", issuer: "https://issuer", audience: "aud"},
		{name: "endpoint invalid", endpoint: "not-a-url", clientID: "id", clientSecret: "secret", issuer: "https://issuer", audience: "aud"},
		{name: "client id", endpoint: "http://zitadel-auth-internal/introspect", clientSecret: "secret", issuer: "https://issuer", audience: "aud"},
		{name: "client secret", endpoint: "http://zitadel-auth-internal/introspect", clientID: "id", issuer: "https://issuer", audience: "aud"},
		{name: "issuer", endpoint: "http://zitadel-auth-internal/introspect", clientID: "id", clientSecret: "secret", audience: "aud"},
		{name: "audience", endpoint: "http://zitadel-auth-internal/introspect", clientID: "id", clientSecret: "secret", issuer: "https://issuer"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewIntegratedOIDCVerifier(tc.endpoint, "", tc.clientID, tc.clientSecret, tc.issuer, tc.audience); err == nil {
				t.Fatal("expected invalid configuration rejection")
			}
		})
	}
}

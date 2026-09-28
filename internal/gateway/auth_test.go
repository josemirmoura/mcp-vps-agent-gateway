package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
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


func TestIntegratedOIDCVerifierUsesPrivateUserInfoBackchannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oidc/v1/userinfo" {
			http.NotFound(w, r)
			return
		}
		if r.Host != "mcp.example.com" || r.Header.Get("X-Forwarded-Proto") != "https" {
			http.Error(w, "bad forwarded identity", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer good-token" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"operator-123"}`))
	}))
	defer server.Close()

	verifier, err := NewIntegratedOIDCVerifier(server.URL+"/oidc/v1/userinfo", "mcp.example.com")
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

	if _, err := verifier(context.Background(), "bad-token", httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
		t.Fatal("expected invalid token rejection")
	}
}

func TestIntegratedOIDCVerifierRejectsInvalidBackchannelURL(t *testing.T) {
	for _, raw := range []string{"", "not-a-url", "ftp://zitadel-auth-internal/userinfo"} {
		if _, err := NewIntegratedOIDCVerifier(raw, "mcp.example.com"); err == nil {
			t.Fatalf("expected invalid URL rejection for %q", raw)
		}
	}
}

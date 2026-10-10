package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

// This is an end-to-end RESOURCE SERVER test against the actual Gateway HTTP
// handler, using disposable localhost introspection. No real issuer, Broker,
// privileged socket, operator token, or network authority is involved.
// It is not a substitute for the official MCP conformance requirements suite.
type oauthResourceProbeExecutor struct {
	mu      sync.Mutex
	calls   int
	subject string
}

func (e *oauthResourceProbeExecutor) Call(ctx context.Context, req wire.Request) (wire.Response, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	e.subject = req.Subject
	if req.Tool != "system.info" {
		return wire.ErrorResponse(req.ID, "unexpected_tool", req.Tool), nil
	}
	return wire.Response{
		ID: req.ID, OK: true,
		Result: []byte(`{"hostname":"disposable-oauth-fixture"}`),
	}, nil
}

func (e *oauthResourceProbeExecutor) snapshot() (int, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls, e.subject
}

func TestIntegratedOAuthResourceServerBoundaryE2E(t *testing.T) {
	const (
		issuer          = "https://issuer.example.invalid"
		resource        = "https://mcp.example.invalid/mcp"
		metadataURL     = "https://mcp.example.invalid/.well-known/oauth-protected-resource"
		audience        = "resource-project-test"
		requiredScope   = "portico.read"
		introspectionID = "disposable-introspector"
		introspectionPW = "disposable-fixture-password"
	)
	introspection := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/introspect" {
			http.NotFound(w, r)
			return
		}
		user, password, ok := r.BasicAuth()
		if !ok || user != introspectionID || password != introspectionPW {
			http.Error(w, "invalid fixture client", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.ParseForm() != nil {
			http.Error(w, "invalid fixture form", http.StatusBadRequest)
			return
		}
		token := r.PostForm.Get("token")
		response := map[string]any{
			"active": true,
			"sub":    "operator-test-subject",
			"scope":  requiredScope,
			"exp":    time.Now().Add(5 * time.Minute).Unix(),
			"iss":    issuer,
			"aud":    []string{audience},
		}
		switch token {
		case "active":
		case "inactive":
			response["active"] = false
		case "expired":
			response["exp"] = time.Now().Add(-time.Minute).Unix()
		case "wrong-issuer":
			response["iss"] = "https://different.example.invalid"
		case "missing-issuer":
			delete(response, "iss")
		case "wrong-audience":
			response["aud"] = []string{"another-resource"}
		case "missing-audience":
			delete(response, "aud")
		case "malformed-audience":
			response["aud"] = 42
		case "wrong-scope":
			response["scope"] = "unrelated.scope"
		case "missing-scope":
			delete(response, "scope")
		case "scope-prefix-only":
			response["scope"] = requiredScope + ".extra"
		case "missing-expiration":
			delete(response, "exp")
		case "empty-subject":
			response["sub"] = ""
		default:
			response = map[string]any{"active": false}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer introspection.Close()

	verifier, err := NewIntegratedOIDCVerifier(
		introspection.URL+"/introspect", "", introspectionID, introspectionPW, issuer, audience,
	)
	if err != nil {
		t.Fatal(err)
	}
	probe := &oauthResourceProbeExecutor{}
	gateway := httptest.NewServer(Handler(probe, AuthConfig{
		Mode:                 "integrated",
		Verifier:             verifier,
		RequiredScopes:       []string{requiredScope},
		ResourceMetadataURL:  metadataURL,
		ResourceIdentifier:   resource,
		AuthorizationServers: []string{issuer},
	}))
	defer gateway.Close()
	httpClient := &http.Client{Timeout: 5 * time.Second}

	t.Run("protected resource metadata is publicly discoverable", func(t *testing.T) {
		resp, err := httpClient.Get(gateway.URL + "/.well-known/oauth-protected-resource")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("metadata HTTP status=%d", resp.StatusCode)
		}
		var metadata struct {
			Resource             string   `json:"resource"`
			AuthorizationServers []string `json:"authorization_servers"`
			BearerMethods        []string `json:"bearer_methods_supported"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
			t.Fatal(err)
		}
		if metadata.Resource != resource || len(metadata.AuthorizationServers) != 1 ||
			metadata.AuthorizationServers[0] != issuer || len(metadata.BearerMethods) != 1 ||
			metadata.BearerMethods[0] != "header" {
			t.Fatalf("invalid protected resource metadata: %+v", metadata)
		}
	})

	// 2026-07-28 Streamable HTTP has no protocol session. A standalone
	// tools/call request with per-request _meta and matching headers exercises
	// the actual Gateway handler and its authentication/subject boundary.
	requestBody := []byte(`{
		"jsonrpc":"2.0","id":1,"method":"tools/call",
		"params":{"name":"system.info","arguments":{},
			"_meta":{
				"io.modelcontextprotocol/protocolVersion":"2026-07-28",
				"io.modelcontextprotocol/clientInfo":{"name":"portico-oauth-smoke","version":"1.0.0"},
				"io.modelcontextprotocol/clientCapabilities":{}
			}
		}
	}`)
	send := func(token string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodPost, gateway.URL+"/mcp", strings.NewReader(string(requestBody)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2026-07-28")
		req.Header.Set("Mcp-Method", "tools/call")
		req.Header.Set("Mcp-Name", "system.info")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return httpClient.Do(req)
	}
	for _, tc := range []struct {
		name, token string
		want        int
	}{
		{name: "missing bearer", token: "", want: http.StatusUnauthorized},
		{name: "unknown bearer", token: "unknown", want: http.StatusUnauthorized},
		{name: "inactive bearer", token: "inactive", want: http.StatusUnauthorized},
		{name: "expired bearer", token: "expired", want: http.StatusUnauthorized},
		{name: "wrong issuer", token: "wrong-issuer", want: http.StatusUnauthorized},
		{name: "missing issuer", token: "missing-issuer", want: http.StatusUnauthorized},
		{name: "wrong audience", token: "wrong-audience", want: http.StatusUnauthorized},
		{name: "missing audience", token: "missing-audience", want: http.StatusUnauthorized},
		{name: "malformed audience", token: "malformed-audience", want: http.StatusUnauthorized},
		{name: "missing expiration", token: "missing-expiration", want: http.StatusUnauthorized},
		{name: "empty subject", token: "empty-subject", want: http.StatusUnauthorized},
		// OAuth scopes are exact tokens, not prefixes; no false authorization.
		// Either 401 or 403 is acceptable here; the SDK controls the status.
		{name: "insufficient scope", token: "wrong-scope", want: 0},
		{name: "missing scope", token: "missing-scope", want: 0},
		{name: "scope prefix confusion", token: "scope-prefix-only", want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, _ := probe.snapshot()
			resp, err := send(tc.token)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			if tc.want != 0 && resp.StatusCode != tc.want {
				t.Fatalf("status=%d want=%d", resp.StatusCode, tc.want)
			}
			if tc.want == 0 && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
				t.Fatalf("insufficient scope status=%d (expected 401 or 403)", resp.StatusCode)
			}
			if after, _ := probe.snapshot(); after != before {
				t.Fatal("unauthorized request reached the tool executor")
			}
			if tc.name == "missing bearer" {
				challenge := resp.Header.Get("WWW-Authenticate")
				if !strings.Contains(strings.ToLower(challenge), "bearer") ||
					!strings.Contains(challenge, fmt.Sprintf(`resource_metadata="%s"`, metadataURL)) {
					t.Fatalf("missing OAuth resource metadata challenge: %q", challenge)
				}
			}
		})
	}

	t.Run("valid bearer executes as the introspected subject", func(t *testing.T) {
		resp, err := send("active")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			// Do not include the body: authentication/protocol error text
			// might contain sensitive values in future SDK versions.
			t.Fatalf("valid stateless MCP call status=%d want=200", resp.StatusCode)
		}
		var rpc struct {
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rpc); err != nil {
			t.Fatal(err)
		}
		if len(rpc.Error) != 0 || len(rpc.Result) == 0 {
			t.Fatal("valid authenticated call did not return a JSON-RPC result")
		}
		if calls, subject := probe.snapshot(); calls != 1 || subject != "operator-test-subject" {
			t.Fatalf("executor calls=%d subject=%q (expected exactly one authenticated subject)", calls, subject)
		}
	})
}

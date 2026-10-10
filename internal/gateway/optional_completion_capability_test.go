package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
)

// The frozen 2026-07-28 conformance suite requires completion/complete even
// when a minimal resource server does not advertise completions. The protocol
// must not falsely advertise the optional feature or execute it implicitly.
// This test does NOT turn the upstream frozen FAIL into a conformance PASS.
func TestMCPCompletionCapabilityIsAbsentAndMethodIsClosed(t *testing.T) {
	root := t.TempDir()
	files, err := securefs.New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(LocalExecutor{FS: files}, AuthConfig{
		Mode: "none", StaticSubject: "conformance-capability-check",
	}))
	defer srv.Close()

	metadata := map[string]any{
		"io.modelcontextprotocol/protocolVersion": "2026-07-28",
		"io.modelcontextprotocol/clientInfo": map[string]string{
			"name": "portico-optional-capability-test", "version": "1",
		},
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}
	send := func(method string, params map[string]any, name string) (int, map[string]json.RawMessage) {
		t.Helper()
		params["_meta"] = metadata
		raw, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
		})
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2026-07-28")
		req.Header.Set("Mcp-Method", method)
		if name != "" {
			req.Header.Set("Mcp-Name", name)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			t.Fatal(err)
		}
		// The Gateway production transport can use JSON or a one-event SSE
		// response; do not treat SSE framing as a feature difference.
		var rpc map[string]json.RawMessage
		if err := json.Unmarshal(body, &rpc); err != nil {
			var decoded []byte
			for _, line := range bytes.Split(body, []byte("\n")) {
				if bytes.HasPrefix(line, []byte("data:")) {
					decoded = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
					break
				}
			}
			if err := json.Unmarshal(decoded, &rpc); err != nil {
				t.Fatalf("expected JSON-RPC response for %s (HTTP %d)", method, resp.StatusCode)
			}
		}
		return resp.StatusCode, rpc
	}

	status, discovery := send("server/discover", map[string]any{}, "")
	if status != http.StatusOK {
		t.Fatalf("discovery returned HTTP %d; expected 200", status)
	}
	if len(discovery["error"]) > 0 {
		t.Fatalf("server/discover rejected valid 2026 metadata")
	}
	var result struct {
		Capabilities map[string]json.RawMessage `json:"capabilities"`
		SupportedVersions []string `json:"supportedVersions"`
	}
	if err := json.Unmarshal(discovery["result"], &result); err != nil {
		t.Fatal(err)
	}
	if len(result.SupportedVersions) == 0 {
		t.Fatal("server/discover has no supported protocol versions")
	}
	for _, v := range result.SupportedVersions {
		if v == "2026-07-28" {
			break
		}
	}
	if _, advertised := result.Capabilities["completions"]; advertised {
		t.Fatal("Gateway advertises completions without an implementation")
	}
	if _, advertised := result.Capabilities["tools"]; !advertised {
		t.Fatal("Gateway no longer advertises its required MCP tool catalog")
	}

	// A validly shaped request must fail closed, not return fake completion
	// values, even when the frozen upstream fixture requires this method.
	status, completion := send("completion/complete", map[string]any{
		"ref": map[string]string{"type": "ref/prompt", "name": "nonexistent"},
		"argument": map[string]string{"name": "x", "value": "y"},
	}, "")
	if status != http.StatusNotFound {
		t.Fatalf("unadvertised completion method returned HTTP %d; expected 404", status)
	}
	if len(completion["result"]) != 0 {
		t.Fatal("unadvertised completion unexpectedly returned a result")
	}
	var rpcErr struct{ Code int `json:"code"` }
	if err := json.Unmarshal(completion["error"], &rpcErr); err != nil {
		t.Fatalf("unadvertised method did not return JSON-RPC error: %v", err)
	}
	if rpcErr.Code != -32601 {
		t.Fatalf("unadvertised method error code=%d; expected -32601", rpcErr.Code)
	}
}

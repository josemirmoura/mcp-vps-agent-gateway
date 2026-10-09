package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	portico "github.com/josemirmoura/mcp-vps-agent-gateway"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
)

// Exercise the real HTTP MCP initialization response, rather than only
// inspecting the Go implementation's in-memory version.
func TestMCPInitializeAdvertisesPackagedVersion(t *testing.T) {
	root := t.TempDir()
	fs, err := securefs.New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(Handler(LocalExecutor{FS: fs}, AuthConfig{Mode: "none", StaticSubject: "version-test"}))
	defer server.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"version-test-client","version":"1"}}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("MCP initialization: HTTP %d: %s", resp.StatusCode, raw)
	}

	payload := strings.TrimSpace(string(raw))
	// Streamable HTTP may carry JSON-RPC data as a single SSE event.
	if strings.HasPrefix(payload, "event:") || strings.HasPrefix(payload, "data:") {
		for _, line := range strings.Split(payload, "\n") {
			if strings.HasPrefix(line, "data:") {
				payload = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				break
			}
		}
	}
	var message struct {
		Result struct {
			ServerInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(payload), &message); err != nil {
		t.Fatalf("invalid MCP initialize response: %v: %s", err, raw)
	}
	if got, want := message.Result.ServerInfo.Version, portico.Version(); got != want {
		t.Fatalf("MCP serverInfo.version=%q; packaged version=%q", got, want)
	}
	if message.Result.ServerInfo.Name != "portico-mcp" {
		t.Fatalf("unexpected MCP server name: %q", message.Result.ServerInfo.Name)
	}
	if message.Result.ProtocolVersion == "" || message.Result.ProtocolVersion == portico.Version() {
		t.Fatalf("MCP protocol revision must be independent of product version: %q", message.Result.ProtocolVersion)
	}
}

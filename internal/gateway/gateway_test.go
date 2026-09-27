package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"path/filepath"
	"testing"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPGate0RoundTrip(t *testing.T) {
	root := t.TempDir()
	fs, err := securefs.New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(Handler(LocalExecutor{FS: fs}, AuthConfig{Mode: "none", StaticSubject: "test-user"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	target := filepath.Join(root, "hello.txt")
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "file.write_test",
		Arguments: map[string]any{"path": target, "content": "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("write tool returned error: %+v", res.Content)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "hello" {
		t.Fatalf("file=%q err=%v", got, err)
	}

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "file.read_test",
		Arguments: map[string]any{"path": target},
	})
	if err != nil || res.IsError {
		t.Fatalf("read failed err=%v result=%+v", err, res)
	}

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "file.read_test",
		Arguments: map[string]any{"path": "/etc/passwd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("forbidden path should return tool error")
	}
}

func TestStaticAuth(t *testing.T) {
	root := t.TempDir()
	fs, _ := securefs.New([]string{root}, []string{root}, 1024)
	h := Handler(LocalExecutor{FS: fs}, AuthConfig{
		Mode: "static", StaticToken: "secret", StaticSubject: "alice",
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}


type instanceExecutor struct{}

func (instanceExecutor) Call(_ context.Context, req wire.Request) (wire.Response, error) {
	if req.Tool != "system.info" {
		return wire.ErrorResponse(req.ID, "unexpected_tool", req.Tool), nil
	}
	raw, _ := json.Marshal(map[string]any{
		"hostname": "host-a",
		"instance_id": "inst-123",
		"instance_name": "VPS Agent | Loja",
	})
	return wire.Response{ID: req.ID, OK: true, Result: raw}, nil
}

func TestSystemInfoPreservesInstanceIdentity(t *testing.T) {
	ts := httptest.NewServer(Handler(instanceExecutor{}, AuthConfig{Mode: "none", StaticSubject: "test-user"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil { t.Fatal(err) }
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "system.info", Arguments: map[string]any{}})
	if err != nil || res.IsError { t.Fatalf("system.info failed: err=%v result=%+v", err, res) }
	raw, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(raw), "inst-123") || !strings.Contains(string(raw), "VPS Agent | Loja") {
		t.Fatalf("instance identity missing: %s", raw)
	}
}

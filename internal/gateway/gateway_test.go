package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
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

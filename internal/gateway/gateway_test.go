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


type rootApprovalExecutor struct{}

func (rootApprovalExecutor) Call(_ context.Context, req wire.Request) (wire.Response, error) {
	switch req.Tool {
	case "permissions.request_root_access":
		raw, _ := json.Marshal(map[string]any{
			"request_id": "apr_test",
			"status": "pending",
			"root": req.Resource,
			"access": "work",
			"delegation_ttl_seconds": 3600,
			"approval_required": true,
			"ceiling_wide": true,
			"physical_ceiling": req.Resource,
			"approval_token": "secret-widget-token",
		})
		return wire.Response{ID: req.ID, OK: true, Result: raw}, nil
	case "permissions.confirm_root_access":
		var in map[string]any
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return wire.ErrorResponse(req.ID, "bad_args", err.Error()), nil
		}
		if in["approval_token"] != "secret-widget-token" {
			return wire.ErrorResponse(req.ID, "permission_denied", "bad token"), nil
		}
		raw, _ := json.Marshal(map[string]any{
			"status": "approved",
			"kind": "root",
		})
		return wire.Response{ID: req.ID, OK: true, Result: raw}, nil
	default:
		return wire.ErrorResponse(req.ID, "unexpected_tool", req.Tool), nil
	}
}

func TestRootApprovalWidgetHidesTokenFromStructuredContent(t *testing.T) {
	ts := httptest.NewServer(Handler(rootApprovalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "permissions.request_root_access",
		Arguments: map[string]any{
			"root": "/opt/project-a",
			"access": "work",
			"ttl_seconds": 3600,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("root request failed: err=%v result=%+v", err, res)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if strings.Contains(string(raw), "secret-widget-token") || strings.Contains(string(raw), "approval_token") {
		t.Fatalf("approval token leaked to model-visible structured content: %s", raw)
	}
	if got, _ := res.Meta["vps-agent/approvalToken"].(string); got != "secret-widget-token" {
		t.Fatalf("hidden widget token missing from result metadata: %#v", res.Meta)
	}

	resource, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: rootApprovalWidgetURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(resource.Contents) != 1 || resource.Contents[0].MIMEType != "text/html;profile=mcp-app" {
		t.Fatalf("unexpected approval widget resource: %+v", resource.Contents)
	}
	if !strings.Contains(resource.Contents[0].Text, "permissions.confirm_root_access") ||
		!strings.Contains(resource.Contents[0].Text, "Autorizar") {
		t.Fatal("approval widget is missing its secure confirmation action")
	}
	html := resource.Contents[0].Text
	for _, marker := range []string{
		"@media (prefers-color-scheme: dark)",
		"--bg: #ffffff",
		"--text: #17191d",
		"--primary: #0b57d0",
		"ceiling_wide",
		"Autorizar todo ",
		"overflow-wrap: anywhere",
		"button:focus-visible",
	} {
		if !strings.Contains(html, marker) {
			t.Fatalf("approval widget accessibility/ceiling marker missing: %q", marker)
		}
	}
	for _, forbidden := range []string{"currentColor", "color: var(--color-text-primary, inherit)"} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("approval widget still depends on unsafe host-theme fallback: %q", forbidden)
		}
	}
}

func TestRootApprovalConfirmToolIsAppOnly(t *testing.T) {
	ts := httptest.NewServer(Handler(rootApprovalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range result.Tools {
		if tool.Name != "permissions.confirm_root_access" {
			continue
		}
		raw, _ := json.Marshal(tool.Meta)
		if !strings.Contains(string(raw), "\"visibility\":[\"app\"]") {
			t.Fatalf("confirm tool is not app-only: %s", raw)
		}
		return
	}
	t.Fatal("permissions.confirm_root_access tool not found")
}

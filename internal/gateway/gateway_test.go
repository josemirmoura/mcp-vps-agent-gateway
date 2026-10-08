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

	portico "github.com/josemirmoura/mcp-vps-agent-gateway"
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
	if !strings.Contains(string(raw), `"gateway_version":"`+portico.Version()+`"`) {
		t.Fatalf("canonical gateway version missing: %s", raw)
	}
	if !strings.Contains(string(raw), `"approval_protocol":"native-mcp-elicitation-when-supported"`) {
		t.Fatalf("approval transport indicator missing: %s", raw)
	}
}


type approvalExecutor struct{}

func (approvalExecutor) Call(_ context.Context, req wire.Request) (wire.Response, error) {
	switch req.Tool {
	case "permissions.request_root_access":
		raw, _ := json.Marshal(map[string]any{
			"request_id": "apr_root_test",
			"status": "pending",
			"kind": "root",
			"root": req.Resource,
			"access": "work",
			"delegation_ttl_seconds": 3600,
			"approval_required": true,
			"ceiling_wide": true,
			"physical_ceiling": req.Resource,
			"approval_token": "secret-root-token",
		})
		return wire.Response{ID: req.ID, OK: true, Result: raw}, nil
	case "permissions.confirm_root_access":
		return approvalConfirmation(req, "secret-root-token", "root")
	case "permissions.request_sensitive_access":
		raw, _ := json.Marshal(map[string]any{
			"request_id": "apr_sensitive_test",
			"status": "pending",
			"kind": "sensitive",
			"path": req.Resource,
			"access": "read",
			"delegation_ttl_seconds": 600,
			"approval_required": true,
			"physical_ceiling": "/opt",
			"approval_token": "secret-sensitive-token",
		})
		return wire.Response{ID: req.ID, OK: true, Result: raw}, nil
	case "permissions.confirm_sensitive_access":
		return approvalConfirmation(req, "secret-sensitive-token", "sensitive")
	default:
		return wire.ErrorResponse(req.ID, "unexpected_tool", req.Tool), nil
	}
}

func approvalConfirmation(req wire.Request, expectedToken, kind string) (wire.Response, error) {
	var in map[string]any
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return wire.ErrorResponse(req.ID, "bad_args", err.Error()), nil
	}
	if in["approval_token"] != expectedToken {
		return wire.ErrorResponse(req.ID, "permission_denied", "bad token"), nil
	}
	decision, _ := in["decision"].(string)
	status := "denied"
	if decision == "approve" {
		status = "approved"
	}
	raw, _ := json.Marshal(map[string]any{
		"status": status,
		"kind": kind,
	})
	return wire.Response{ID: req.ID, OK: true, Result: raw}, nil
}

func TestRootAccessUsesNativeMCPApproval(t *testing.T) {
	t.Setenv("VPS_AGENT_LANG", "pt-BR")
	ts := httptest.NewServer(Handler(approvalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()

	var prompt string
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, &mcp.ClientOptions{
		ElicitationHandler: func(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			prompt = req.Params.Message
			return &mcp.ElicitResult{Action: "accept"}, nil
		},
	})
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "permissions.request_root_access",
		Arguments: map[string]any{
			"root": "/opt",
			"access": "work",
			"ttl_seconds": 3600,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("native root approval failed: err=%v result=%+v", err, res)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(raw), "approved") {
		t.Fatalf("expected approved final result, got %s", raw)
	}
	if strings.Contains(string(raw), "secret-root-token") || strings.Contains(string(raw), "approval_token") {
		t.Fatalf("approval token leaked to final model-visible output: %s", raw)
	}
	if !strings.Contains(prompt, "Autorizar Pórtico?") ||
		!strings.Contains(prompt, "ATENÇÃO") ||
		!strings.Contains(prompt, "/opt") ||
		!strings.Contains(prompt, "Perfil:") ||
		!strings.Contains(prompt, "Duração:") {
		t.Fatalf("native approval prompt is missing operator-facing authority context: %q", prompt)
	}
	if len(prompt) > 220 {
		t.Fatalf("native approval prompt is too long for compact mobile rendering: %d bytes: %q", len(prompt), prompt)
	}
}

func TestRootAccessNativeDeclineKeepsAuthorityLocked(t *testing.T) {
	ts := httptest.NewServer(Handler(approvalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, &mcp.ClientOptions{
		ElicitationHandler: func(_ context.Context, _ *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "decline"}, nil
		},
	})
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "permissions.request_root_access",
		Arguments: map[string]any{"root": "/opt/project-a", "access": "read"},
	})
	if err != nil || res.IsError {
		t.Fatalf("decline flow failed: err=%v result=%+v", err, res)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(raw), "denied") {
		t.Fatalf("expected denied final result, got %s", raw)
	}
}

func TestSensitiveAccessUsesNativeMCPApproval(t *testing.T) {
	ts := httptest.NewServer(Handler(approvalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()

	var prompt string
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, &mcp.ClientOptions{
		ElicitationHandler: func(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			prompt = req.Params.Message
			return &mcp.ElicitResult{Action: "accept"}, nil
		},
	})
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "permissions.request_sensitive_access",
		Arguments: map[string]any{
			"path": "/opt/project-a/.env",
			"access": "read",
			"ttl_seconds": 600,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("native protected-file approval failed: err=%v result=%+v", err, res)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(raw), "approved") {
		t.Fatalf("expected approved final result, got %s", raw)
	}
	if strings.Contains(string(raw), "secret-sensitive-token") || !strings.Contains(prompt, ".env") {
		t.Fatalf("protected-file native approval leaked token or lost context: result=%s prompt=%q", raw, prompt)
	}
}

func TestApprovalPromptsStayCompactForMobile(t *testing.T) {
	t.Setenv("VPS_AGENT_LANG", "pt-BR")

	cases := []struct {
		name   string
		kind   string
		values map[string]any
		want   []string
	}{
		{
			name: "root",
			kind: "root",
			values: map[string]any{
				"root":                   "/opt/project-a",
				"access":                 "work",
				"delegation_ttl_seconds": 3600,
				"physical_ceiling":       "/opt",
				"ceiling_wide":           false,
			},
			want: []string{"Autorizar Pórtico?", "/opt/project-a", "Perfil:", "Duração:"},
		},
		{
			name: "protected-file",
			kind: "sensitive",
			values: map[string]any{
				"path":                   "/opt/project-a/.env",
				"access":                 "read",
				"delegation_ttl_seconds": 600,
				"physical_ceiling":       "/opt",
			},
			want: []string{"Autorizar arquivo protegido?", "/opt/project-a/.env", "Perfil:", "Duração:"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prompt := approvalMessage(tc.kind, tc.values)
			if len(prompt) > 220 {
				t.Fatalf("approval prompt is too long for compact mobile rendering: %d bytes: %q", len(prompt), prompt)
			}
			if strings.Count(prompt, "\n") > 5 {
				t.Fatalf("approval prompt has too many lines for compact mobile rendering: %q", prompt)
			}
			for _, want := range tc.want {
				if !strings.Contains(prompt, want) {
					t.Fatalf("approval prompt missing %q: %q", want, prompt)
				}
			}
		})
	}
}

func TestApprovalFallsBackWithoutElicitationAndHidesConfirmTools(t *testing.T) {
	ts := httptest.NewServer(Handler(approvalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "permissions.request_root_access",
		Arguments: map[string]any{"root": "/opt/project-a", "access": "read"},
	})
	if err != nil || res.IsError {
		t.Fatalf("fallback root request failed: err=%v result=%+v", err, res)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(raw), "operator_fallback") {
		t.Fatalf("expected explicit operator fallback, got %s", raw)
	}
	if !strings.Contains(string(raw), "operator_approval_guide") {
		t.Fatalf("fallback must explain trusted operator approval path: %s", raw)
	}
	if strings.Contains(string(raw), "secret-root-token") || strings.Contains(string(raw), "approval_token") {
		t.Fatalf("fallback leaked approval token: %s", raw)
	}

	toolsResult, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range toolsResult.Tools {
		if tool.Name == "permissions.confirm_root_access" || tool.Name == "permissions.confirm_sensitive_access" {
			t.Fatalf("internal confirmation tool must not be exposed to the model/client tool catalog: %s", tool.Name)
		}
	}
}

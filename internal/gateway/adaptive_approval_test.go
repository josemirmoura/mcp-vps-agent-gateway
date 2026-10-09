package gateway

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestApprovalCapabilityNegotiation(t *testing.T) {
	cases := []struct {
		name string
		caps *mcp.ClientCapabilities
		want approvalClientSupport
	}{
		{"absent", nil, approvalClientSupport{}},
		{"legacy-form", &mcp.ClientCapabilities{Elicitation: &mcp.ElicitationCapabilities{}}, approvalClientSupport{Form: true}},
		{"url-only", &mcp.ClientCapabilities{Elicitation: &mcp.ElicitationCapabilities{URL: &mcp.URLElicitationCapabilities{}}}, approvalClientSupport{URL: true}},
		{"wrong-mime", &mcp.ClientCapabilities{Extensions: map[string]any{approvalExtension: map[string]any{"mimeTypes": []string{"text/html"}}}}, approvalClientSupport{}},
		{"malformed", &mcp.ClientCapabilities{Extensions: map[string]any{approvalExtension: "yes"}}, approvalClientSupport{}},
		{"apps", &mcp.ClientCapabilities{Extensions: map[string]any{approvalExtension: map[string]any{"mimeTypes": []string{approvalAppMIME}}}}, approvalClientSupport{Apps: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := approvalCapabilities(tc.caps); got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestMCPAppsResourceAndTextFallbackSharePersistentRequest(t *testing.T) {
	t.Setenv("VPS_AGENT_MCP_APPS", "1")
	t.Setenv("VPS_AGENT_OPERATOR_PORTAL_URL", "https://operator.example.test/operator")
	t.Setenv("PORTICO_OPERATOR_FRAME_ANCESTORS", "https://host.example.test")
	ts := httptest.NewServer(Handler(approvalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "alice"}))
	defer ts.Close()
	for _, apps := range []bool{false, true} {
		options := &mcp.ClientOptions{}
		if apps {
			options.Capabilities = &mcp.ClientCapabilities{Extensions: map[string]any{approvalExtension: map[string]any{"mimeTypes": []string{approvalAppMIME}}}}
		}
		client := mcp.NewClient(&mcp.Implementation{Name: "same-name-untrusted-client", Version: "v1"}, options)
		session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "permissions.request_root_access", Arguments: map[string]any{"root": "/opt/project-a", "access": "read", "ttl_seconds": 300}})
		if err != nil || res.IsError {
			t.Fatalf("request failed %v %+v", err, res)
		}
		raw, _ := json.Marshal(res.StructuredContent)
		expected := "operator_web"
		if apps {
			expected = "mcp_apps"
		}
		if !strings.Contains(string(raw), expected) || strings.Contains(string(raw), "approval_token") {
			t.Fatalf("unexpected adaptive output: %s", raw)
		}
		if apps {
			resource, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: approvalAppURI})
			if err != nil {
				t.Fatal(err)
			}
			if len(resource.Contents) != 1 || resource.Contents[0].MIMEType != approvalAppMIME {
				t.Fatal("invalid resource")
			}
			text := resource.Contents[0].Text
			if strings.Contains(text, "__PORTICO_CONFIG__") || strings.Contains(text, "approval_token") || !strings.Contains(text, "https://operator.example.test") {
				t.Fatal("resource not configured or credential leaked")
			}
		}
		session.Close()
	}
}

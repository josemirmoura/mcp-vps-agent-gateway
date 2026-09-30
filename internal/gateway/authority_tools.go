package gateway

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type sensitiveAccessRequestInput struct {
	Path        string `json:"path" jsonschema:"exact protected file path inside an already authorized root"`
	Access      string `json:"access" jsonschema:"temporary protected-file profile: read or work"`
	TTLSeconds  int64  `json:"ttl_seconds,omitempty" jsonschema:"temporary access lifetime in seconds; 0 uses the safe default"`
	OperationID string `json:"operation_id,omitempty" jsonschema:"stable retry identity when the client can preserve one"`
}

type sensitiveAccessRevokeInput struct {
	Path        string `json:"path" jsonschema:"exact protected file path whose temporary access should be revoked"`
	OperationID string `json:"operation_id,omitempty" jsonschema:"stable retry identity when the client can preserve one"`
}

type sensitiveAccessConfirmInput struct {
	RequestID     string `json:"request_id" jsonschema:"pending protected-file approval request identifier"`
	ApprovalToken string `json:"approval_token" jsonschema:"one-time approval token supplied only to the approval UI"`
	Decision      string `json:"decision" jsonschema:"approve or deny"`
}

func registerAuthorityTools(server *mcp.Server, s *Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "permissions.discover_scope",
		Description: "List only the immediate directory names under the configured physical filesystem ceiling. This is discovery-only: directory contents remain locked until separately authorized.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true, DestructiveHint: boolPtr(false), IdempotentHint: true, OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in limitInput) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		args, _ := json.Marshal(map[string]any{"limit": in.Limit})
		if err := s.call(ctx, "permissions.discover_scope", "", "discover", args, &out, false, ""); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "permissions.request_sensitive_access",
		Description: "Request temporary access to one protected secret file such as .env inside an already-authorized root. Normal root delegation never unlocks protected secrets.",
		Meta: mcp.Meta{
			"ui": map[string]any{
				"resourceUri": rootApprovalWidgetURI,
				"visibility":  []string{"model", "app"},
			},
			"openai/outputTemplate":          rootApprovalWidgetURI,
			"openai/toolInvocation/invoking": "Preparing protected-file request…",
			"openai/toolInvocation/invoked":  "Protected-file request ready.",
		},
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: true, OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sensitiveAccessRequestInput) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		args, _ := json.Marshal(in)
		if err := s.call(ctx, "permissions.request_sensitive_access", in.Path, "request", args, &out, true, in.OperationID); err != nil {
			return nil, out, err
		}
		approvalToken, _ := out["approval_token"].(string)
		delete(out, "approval_token")
		return &mcp.CallToolResult{Meta: mcp.Meta{
			"vps-agent/approvalToken": approvalToken,
		}}, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "permissions.confirm_sensitive_access",
		Description: "Approve or deny a pending protected-file grant from the interactive approval card. This app-only tool requires a one-time token hidden from the model.",
		Meta: mcp.Meta{
			"ui": map[string]any{"visibility": []string{"app"}},
			"openai/widgetAccessible": true,
			"openai/visibility":       "private",
		},
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: true, OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sensitiveAccessConfirmInput) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		args, _ := json.Marshal(in)
		if err := s.call(ctx, "permissions.confirm_sensitive_access", in.RequestID, in.Decision, args, &out, true, in.RequestID+":"+in.Decision); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "permissions.list_sensitive_access",
		Description: "List this authenticated subject's active temporary protected-file grants.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true, DestructiveHint: boolPtr(false), IdempotentHint: true, OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		if err := s.call(ctx, "permissions.list_sensitive_access", "", "list", nil, &out, false, ""); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "permissions.revoke_sensitive_access",
		Description: "Immediately revoke this authenticated subject's temporary access to one protected secret path.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: true, OpenWorldHint: boolPtr(false),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sensitiveAccessRevokeInput) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		args, _ := json.Marshal(in)
		if err := s.call(ctx, "permissions.revoke_sensitive_access", in.Path, "revoke", args, &out, true, in.OperationID); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})
}

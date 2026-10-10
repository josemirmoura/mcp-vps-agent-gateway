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

func registerAuthorityTools(server *mcp.Server, s *Server) {
	mcp.AddTool(server, annotatedTool("permissions.discover_scope", "List only the immediate directory names under the configured physical filesystem ceiling. This is discovery-only: directory contents remain locked until separately authorized."), func(ctx context.Context, _ *mcp.CallToolRequest, in limitInput) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		args, _ := json.Marshal(map[string]any{"limit": in.Limit})
		if err := s.call(ctx, "permissions.discover_scope", "", "discover", args, &out, false, ""); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, approvalTool("permissions.request_sensitive_access", "Request temporary access to one protected file. Operator session and request-bound step-up are required; client acceptance alone never grants access."), func(ctx context.Context, req *mcp.CallToolRequest, in sensitiveAccessRequestInput) (*mcp.CallToolResult, any, error) {
		if out, handled, err := s.navigationContinuation(ctx, req, "sensitive", in.Path); handled {
			return nil, out, err
		}
		var out map[string]any
		args, _ := json.Marshal(in)
		if err := s.call(ctx, "permissions.request_sensitive_access", in.Path, "request", args, &out, true, in.OperationID); err != nil {
			return nil, out, err
		}
		return s.presentApproval(ctx, req, "sensitive", out)
	})

	mcp.AddTool(server, annotatedTool("permissions.list_sensitive_access", "List this authenticated subject's active temporary protected-file grants."), func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		if err := s.call(ctx, "permissions.list_sensitive_access", "", "list", nil, &out, false, ""); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, annotatedTool("permissions.revoke_sensitive_access", "Immediately revoke this authenticated subject's temporary access to one protected secret path."), func(ctx context.Context, _ *mcp.CallToolRequest, in sensitiveAccessRevokeInput) (*mcp.CallToolResult, map[string]any, error) {
		var out map[string]any
		args, _ := json.Marshal(in)
		if err := s.call(ctx, "permissions.revoke_sensitive_access", in.Path, "revoke", args, &out, true, in.OperationID); err != nil {
			return nil, out, err
		}
		return nil, out, nil
	})
}

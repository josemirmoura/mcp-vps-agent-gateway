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

	mcp.AddTool(server, annotatedTool("permissions.request_sensitive_access", "Request temporary operator-approved access to one protected secret file such as .env inside an already-authorized root. Normal root delegation never unlocks protected secrets. On clients with MCP elicitation support, the host renders its native confirmation UI."), func(ctx context.Context, req *mcp.CallToolRequest, in sensitiveAccessRequestInput) (*mcp.CallToolResult, any, error) {
		if state, decision, handled, err := nativeApprovalDecision(req, "sensitive"); handled {
			if err != nil {
				return nil, nil, err
			}
			var out map[string]any
			args, _ := json.Marshal(map[string]any{
				"request_id": state.RequestID,
				"approval_token": state.ApprovalToken,
				"decision": decision,
			})
			if err := s.call(ctx, "permissions.confirm_sensitive_access", state.RequestID, decision, args, &out, true, state.RequestID+":"+decision); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		}

		var out map[string]any
		args, _ := json.Marshal(in)
		if err := s.call(ctx, "permissions.request_sensitive_access", in.Path, "request", args, &out, true, in.OperationID); err != nil {
			return nil, out, err
		}
		approvalToken, _ := out["approval_token"].(string)
		delete(out, "approval_token")

		if link := operatorWebApprovalLink(out); link != "" {
			out["approval_method"] = "operator_web"
			out["operator_approval_url"] = link
			out["message"] = "Pórtico: solicitação pendente. Abra operator_approval_url em seu navegador, entre como operador e confirme ou negue. O link não concede acesso."
			return nil, out, nil
		}
		if !supportsNativeElicitation(req) {
			out["approval_method"] = "operator_fallback"
			out["message"] = "Pórtico: pedido pendente. Acesse uma sessão SSH autenticada e use scripts/operator-approvals.py para decidir; nunca compartilhe credenciais com a IA."
			out["operator_approval_guide"] = "docs/operator-approval-fallback.md"
			return nil, out, nil
		}
		approval, err := nativeApprovalResult("sensitive", out, approvalToken)
		if err != nil {
			return nil, nil, err
		}
		return approval, nil, nil
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

package broker

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type Broker struct {
	Policy     *policy.Config
	FS         *securefs.Manager
	State      *state.Store
	Services   ServiceManager
	AdminToken string
}

func (b *Broker) Handle(ctx context.Context, req wire.Request) wire.Response {
	if req.ID == "" {
		req.ID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	resp := b.handle(ctx, req)
	decision := "allow"
	if !resp.OK {
		decision = "deny"
	}
	if b.State != nil {
		_, _ = b.State.AppendAudit(ctx, state.AuditEvent{
			Subject: req.Subject, Tool: req.Tool, Resource: req.Resource,
			Decision: decision, ActionID: req.InvocationID,
		})
	}
	return resp
}

func (b *Broker) handle(ctx context.Context, req wire.Request) wire.Response {
	switch req.Tool {
	case "system.info":
		host, _ := os.Hostname()
		return ok(req.ID, map[string]any{
			"hostname": host, "goos": runtime.GOOS, "goarch": runtime.GOARCH,
			"cpus": runtime.NumCPU(),
		})
	case "permissions.status":
		return ok(req.ID, map[string]any{
			"mode":         b.Policy.Mode,
			"full_enabled": b.Policy.Features.FullModeEnabled && b.Policy.Enabled,
		})
	case "file.read", "file.read_test":
		if b.FS == nil {
			return deny(req.ID, "filesystem_unavailable", "filesystem manager is not configured")
		}
		data, err := b.FS.ReadFile(req.Resource)
		if err != nil {
			return deny(req.ID, "permission_denied", err.Error())
		}
		return ok(req.ID, map[string]any{"content": string(data), "bytes": len(data)})
	case "file.write", "file.write_test":
		if b.FS == nil {
			return deny(req.ID, "filesystem_unavailable", "filesystem manager is not configured")
		}
		var in struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "write requires invocation id")
		}
		return b.writeFile(ctx, req, []byte(in.Content))
	case "service.status":
		if !b.Policy.CanService(req.Resource, "status") {
			return deny(req.ID, "permission_denied", "service is outside policy")
		}
		status, err := b.Services.Status(ctx, req.Resource)
		if err != nil {
			return deny(req.ID, "service_error", err.Error())
		}
		return ok(req.ID, map[string]any{"service": req.Resource, "status": status})
	case "service.restart":
		if !b.Policy.CanService(req.Resource, "restart") {
			return deny(req.ID, "permission_denied", "service restart is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "restart requires invocation id")
		}
		return b.restartService(ctx, req)
	case "admin.grant.issue":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		if !b.Policy.Features.FullModeEnabled || !b.Policy.Enabled {
			return deny(req.ID, "full_disabled", "full mode is disabled")
		}
		var in struct {
			Subject      string   `json:"subject"`
			Capabilities []string `json:"capabilities"`
			TTLSeconds   int64    `json:"ttl_seconds"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
		ttl := time.Duration(in.TTLSeconds) * time.Second
		if ttl <= 0 || ttl > b.Policy.MaxGrantTTL() {
			return deny(req.ID, "invalid_ttl", "ttl exceeds policy")
		}
		g, err := b.State.IssueGrant(ctx, in.Subject, in.Capabilities, ttl)
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		return ok(req.ID, g)
	case "admin.revoke_all":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		if err := b.State.RevokeAll(ctx); err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		return ok(req.ID, map[string]any{"revoked": true})
	case "shell.exec_admin":
		if !b.Policy.Features.FullModeEnabled || !b.Policy.Enabled {
			return deny(req.ID, "full_disabled", "full mode is disabled")
		}
		valid, err := b.State.ValidateGrant(ctx, req.GrantID, req.Subject, "shell.admin")
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		if !valid {
			return deny(req.ID, "permission_denied", "valid shell.admin grant required")
		}
		return deny(req.ID, "not_implemented", "admin shell is intentionally not implemented before Gate 5")
	default:
		return deny(req.ID, "unknown_tool", "unknown broker tool")
	}
}

func (b *Broker) adminOK(token string) bool {
	if b.AdminToken == "" || token == "" || len(token) != len(b.AdminToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(b.AdminToken)) == 1
}

func (b *Broker) writeFile(ctx context.Context, req wire.Request, data []byte) wire.Response {
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "resource": req.Resource,
		"content": string(data),
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
	if b.State != nil {
		decision, cached, err := b.State.BeginOperation(ctx, req.InvocationID, req.Subject, req.Tool, requestHash)
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		switch decision {
		case state.OperationCached:
			return wire.Response{ID: req.ID, OK: true, Result: cached}
		case state.OperationConflict:
			return deny(req.ID, "idempotency_conflict", "invocation id reused with different request")
		case state.OperationReconcile:
			return deny(req.ID, "reconcile_required", "previous attempt outcome is uncertain")
		}
	}
	if err := b.FS.WriteFileAtomic(req.Resource, data); err != nil {
		return deny(req.ID, "permission_denied", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"bytes": len(data), "written": true})
	if b.State != nil {
		if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
			return deny(req.ID, "state_error", "write completed but operation journal update failed: "+err.Error())
		}
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func (b *Broker) restartService(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "restart requires durable state")
	}
	requestHash, _ := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "service": req.Resource,
	})
	decision, cached, err := b.State.BeginOperation(ctx, req.InvocationID, req.Subject, req.Tool, requestHash)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	switch decision {
	case state.OperationCached:
		return wire.Response{ID: req.ID, OK: true, Result: cached}
	case state.OperationConflict:
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous restart outcome is uncertain; inspect service before retry")
	}

	lock, err := b.State.AcquireLock(ctx, "service:"+req.Resource, req.InvocationID, 2*time.Minute)
	if err != nil {
		return deny(req.ID, "resource_busy", err.Error())
	}
	defer b.State.ReleaseLock(context.Background(), lock)

	status, err := b.Services.Restart(ctx, req.Resource)
	if err != nil {
		return deny(req.ID, "service_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"service": req.Resource, "status": status, "restarted": true})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "restart completed but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func ok(id string, value any) wire.Response {
	raw, err := json.Marshal(value)
	if err != nil {
		return deny(id, "encode_error", err.Error())
	}
	return wire.Response{ID: id, OK: true, Result: raw}
}

func deny(id, code, message string) wire.Response {
	return wire.ErrorResponse(id, code, message)
}

var _ = errors.New

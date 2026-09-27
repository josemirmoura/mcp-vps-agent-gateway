package broker

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"runtime"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/jobs"
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
	Docker     DockerManager
	Jobs       *jobs.Manager
	AdminToken      string
	ExpectedSubject string
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
	if !strings.HasPrefix(req.Tool, "admin.") && b.ExpectedSubject != "" && req.Subject != b.ExpectedSubject {
		return deny(req.ID, "identity_mismatch", "subject is not authorized for this Broker")
	}
	switch req.Tool {
	case "system.info":
		host, _ := os.Hostname()
		return ok(req.ID, map[string]any{
			"hostname": host, "goos": runtime.GOOS, "goarch": runtime.GOARCH,
			"cpus": runtime.NumCPU(),
		})
	case "system.health":
		auditOK := true
		if b.State != nil {
			auditOK = b.State.VerifyAudit(ctx) == nil
		}
		return ok(req.ID, map[string]any{
			"ok": true, "audit_chain_ok": auditOK,
			"state_configured":  b.State != nil,
			"docker_configured": b.Docker != nil,
			"jobs_configured":   b.Jobs != nil,
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
		if b.Services == nil {
			return deny(req.ID, "service_unavailable", "service manager is not configured")
		}
		if !b.Policy.CanService(req.Resource, "status") {
			return deny(req.ID, "permission_denied", "service is outside policy")
		}
		status, err := b.Services.Status(ctx, req.Resource)
		if err != nil {
			return deny(req.ID, "service_error", err.Error())
		}
		return ok(req.ID, map[string]any{"service": req.Resource, "status": status})
	case "service.restart":
		if b.Services == nil {
			return deny(req.ID, "service_unavailable", "service manager is not configured")
		}
		if !b.Policy.CanService(req.Resource, "restart") {
			return deny(req.ID, "permission_denied", "service restart is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "restart requires invocation id")
		}
		return b.restartService(ctx, req)
	case "docker.inspect":
		if b.Docker == nil {
			return deny(req.ID, "docker_unavailable", "docker manager is not configured")
		}
		if !b.Policy.CanDocker(req.Resource, "inspect") {
			return deny(req.ID, "permission_denied", "docker resource is outside policy")
		}
		info, err := b.Docker.Inspect(ctx, req.Resource)
		if err != nil {
			return deny(req.ID, "docker_error", err.Error())
		}
		return ok(req.ID, info)
	case "docker.logs":
		if b.Docker == nil {
			return deny(req.ID, "docker_unavailable", "docker manager is not configured")
		}
		if !b.Policy.CanDocker(req.Resource, "logs") {
			return deny(req.ID, "permission_denied", "docker logs are outside policy")
		}
		var in struct {
			Lines int `json:"lines"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		logs, err := b.Docker.Logs(ctx, req.Resource, in.Lines)
		if err != nil {
			return deny(req.ID, "docker_error", err.Error())
		}
		return ok(req.ID, map[string]any{"resource": req.Resource, "logs": logs})
	case "docker.action":
		if b.Docker == nil {
			return deny(req.ID, "docker_unavailable", "docker manager is not configured")
		}
		var in struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
		if in.Action != "restart" {
			return deny(req.ID, "unsupported_action", "only restart is implemented")
		}
		if !b.Policy.CanDocker(req.Resource, in.Action) {
			return deny(req.ID, "permission_denied", "docker action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "docker action requires invocation id")
		}
		return b.restartDocker(ctx, req)
	case "job.status":
		if b.Jobs == nil {
			return deny(req.ID, "jobs_unavailable", "job manager is not configured")
		}
		rec, err := b.Jobs.Status(ctx, req.Subject, req.Resource)
		if err != nil {
			return deny(req.ID, "job_error", err.Error())
		}
		return ok(req.ID, rec)
	case "job.tail":
		if b.Jobs == nil {
			return deny(req.ID, "jobs_unavailable", "job manager is not configured")
		}
		var in struct {
			Lines int `json:"lines"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		out, err := b.Jobs.Tail(ctx, req.Subject, req.Resource, in.Lines)
		if err != nil {
			return deny(req.ID, "job_error", err.Error())
		}
		return ok(req.ID, map[string]any{"job_id": req.Resource, "output": out})
	case "job.cancel":
		if b.Jobs == nil {
			return deny(req.ID, "jobs_unavailable", "job manager is not configured")
		}
		if err := b.Jobs.Cancel(ctx, req.Subject, req.Resource); err != nil {
			return deny(req.ID, "job_error", err.Error())
		}
		return ok(req.ID, map[string]any{"job_id": req.Resource, "cancelled": true})
	case "permissions.request_elevation":
		return b.requestElevation(ctx, req)
	case "admin.approval.list":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		pending, err := b.State.ListPendingApprovals(ctx)
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		return ok(req.ID, pending)
	case "admin.approval.approve":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		return b.decideApproval(ctx, req, "approved")
	case "admin.approval.deny":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		return b.decideApproval(ctx, req, "denied")
	case "admin.audit.status":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		status, err := b.State.AuditStatus(ctx)
		if err != nil {
			return deny(req.ID, "audit_error", err.Error())
		}
		return ok(req.ID, status)
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
		return deny(req.ID, "not_implemented", "admin shell stays disabled until Gate 5 environment tests pass")
	default:
		return deny(req.ID, "unknown_tool", "unknown broker tool")
	}
}

func (b *Broker) requestElevation(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "elevation requires durable state")
	}
	if !b.Policy.Features.FullModeEnabled || !b.Policy.Enabled {
		return deny(req.ID, "full_disabled", "full mode is disabled")
	}
	var in struct {
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
	for _, cap := range in.Capabilities {
		if !b.Policy.CanGrant(cap) {
			return deny(req.ID, "permission_denied", "requested capability is not grantable by policy: "+cap)
		}
	}
	a, err := b.State.CreateApproval(ctx, req.Subject, in.Capabilities, ttl, 10*time.Minute)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	return ok(req.ID, map[string]any{
		"request_id": a.ID, "status": a.Status, "expires_at": a.ExpiresAt,
	})
}

func (b *Broker) decideApproval(ctx context.Context, req wire.Request, decision string) wire.Response {
	var in struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	a, err := b.State.DecideApproval(ctx, in.RequestID, decision)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	if decision == "denied" {
		return ok(req.ID, map[string]any{"request_id": a.ID, "status": "denied"})
	}
	g, err := b.State.IssueGrant(ctx, a.Subject, a.Capabilities, a.TTL)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	return ok(req.ID, map[string]any{
		"request_id": a.ID, "status": "approved", "grant_id": g.ID,
		"subject": g.Subject, "capabilities": g.Capabilities, "expires_at": g.ExpiresAt,
	})
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

func (b *Broker) restartDocker(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "docker mutation requires durable state")
	}
	requestHash, _ := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "resource": req.Resource, "action": "restart",
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
		return deny(req.ID, "reconcile_required", "previous docker restart outcome is uncertain; inspect before retry")
	}
	lock, err := b.State.AcquireLock(ctx, "docker:"+req.Resource, req.InvocationID, 2*time.Minute)
	if err != nil {
		return deny(req.ID, "resource_busy", err.Error())
	}
	defer b.State.ReleaseLock(context.Background(), lock)

	info, err := b.Docker.Restart(ctx, req.Resource)
	if err != nil {
		return deny(req.ID, "docker_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"resource": req.Resource, "restarted": true, "inspect": info})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "docker restarted but journal update failed: "+err.Error())
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

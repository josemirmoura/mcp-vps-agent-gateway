package broker

import (
	"context"
	"errors"
	"crypto/subtle"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/hostexec"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/jobs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sandbox"
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
	InstanceID      string
	InstanceName    string
	elevationMu     sync.Mutex
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
		if _, err := b.State.AppendAudit(ctx, state.AuditEvent{
			InstanceID: b.InstanceID, InstanceName: b.InstanceName,
			Subject: req.Subject, Tool: req.Tool, Resource: req.Resource,
			Decision: decision, ActionID: req.InvocationID,
		}); err != nil {
			slog.ErrorContext(ctx, "audit_append_failed",
				"instance_id", b.InstanceID, "instance_name", b.InstanceName,
				"request_id", req.ID, "invocation_id", req.InvocationID,
				"subject", req.Subject, "tool", req.Tool, "resource", req.Resource,
				"error", err)
		}
	}
	slog.InfoContext(ctx, "broker_request",
		"instance_id", b.InstanceID, "instance_name", b.InstanceName,
		"request_id", req.ID, "invocation_id", req.InvocationID,
		"subject", req.Subject, "tool", req.Tool, "resource", req.Resource,
		"decision", decision)
	return resp
}

func (b *Broker) handle(ctx context.Context, req wire.Request) wire.Response {
	if !strings.HasPrefix(req.Tool, "admin.") && b.ExpectedSubject != "" && req.Subject != b.ExpectedSubject {
		return deny(req.ID, "identity_mismatch", "subject is not authorized for this Broker")
	}
	switch req.Tool {
	case "system.info":
		host, _ := os.Hostname()
		if out, err := hostexec.CommandContext(ctx, "hostname").Output(); err == nil && strings.TrimSpace(string(out)) != "" {
			host = strings.TrimSpace(string(out))
		}
		return ok(req.ID, map[string]any{
			"hostname": host, "goos": runtime.GOOS, "goarch": runtime.GOARCH,
			"cpus": runtime.NumCPU(), "instance_id": b.InstanceID, "instance_name": b.InstanceName,
			"physical_scope_root": os.Getenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT"),
			"whole_host": os.Getenv("VPS_AGENT_WHOLE_HOST") == "1",
		})
	case "system.health":
		return ok(req.ID, b.healthSnapshot(ctx))
	case "system.disk":
		if !b.Policy.CanDiagnostic("system.disk") {
			return deny(req.ID, "permission_denied", "disk diagnostics are disabled by policy")
		}
		info, err := diskInfo(ctx)
		if err != nil {
			return deny(req.ID, "diagnostic_error", err.Error())
		}
		return ok(req.ID, map[string]any{"filesystems": info})
	case "system.memory":
		if !b.Policy.CanDiagnostic("system.memory") {
			return deny(req.ID, "permission_denied", "memory diagnostics are disabled by policy")
		}
		info, err := memoryInfo()
		if err != nil {
			return deny(req.ID, "diagnostic_error", err.Error())
		}
		return ok(req.ID, info)
	case "process.list":
		if !b.Policy.CanDiagnostic("process.list") {
			return deny(req.ID, "permission_denied", "process listing is disabled by policy")
		}
		var in struct { Limit int `json:"limit"` }
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		rows, err := processList(ctx, in.Limit)
		if err != nil {
			return deny(req.ID, "diagnostic_error", err.Error())
		}
		return ok(req.ID, map[string]any{"processes": rows})
	case "process.inspect":
		if !b.Policy.CanDiagnostic("process.inspect") {
			return deny(req.ID, "permission_denied", "process inspection is disabled by policy")
		}
		pid, err := strconv.Atoi(req.Resource)
		if err != nil {
			return deny(req.ID, "invalid_args", "resource must be a numeric pid")
		}
		info, err := processInspect(pid)
		if err != nil {
			return deny(req.ID, "diagnostic_error", err.Error())
		}
		return ok(req.ID, info)
	case "network.listen":
		if !b.Policy.CanDiagnostic("network.listen") {
			return deny(req.ID, "permission_denied", "network listener diagnostics are disabled by policy")
		}
		var in struct { Limit int `json:"limit"` }
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		rows, err := listenInfo(ctx, in.Limit)
		if err != nil {
			return deny(req.ID, "diagnostic_error", err.Error())
		}
		return ok(req.ID, map[string]any{"listeners": rows})
	case "network.check":
		if !b.Policy.CanDiagnostic("network.check") {
			return deny(req.ID, "permission_denied", "network checks are disabled by policy")
		}
		if !b.Policy.CanNetworkDestination(req.Resource) {
			return deny(req.ID, "permission_denied", "network destination is outside policy")
		}
		var in struct { TimeoutSeconds int `json:"timeout_seconds"` }
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		result, err := networkCheck(ctx, req.Resource, time.Duration(in.TimeoutSeconds)*time.Second)
		if err != nil {
			return deny(req.ID, "diagnostic_error", err.Error())
		}
		return ok(req.ID, result)
	case "permissions.status":
		return ok(req.ID, map[string]any{
			"mode":         b.Policy.Mode,
			"full_enabled": b.Policy.Features.FullModeEnabled && b.Policy.Enabled,
		})
	case "file.read", "file.read_test":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("read") {
			return deny(req.ID, "permission_denied", "file read is disabled by policy")
		}
		data, err := fs.ReadFile(req.Resource)
		if err != nil {
			return deny(req.ID, "permission_denied", err.Error())
		}
		return ok(req.ID, map[string]any{"content": string(data), "bytes": len(data)})
	case "file.list":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("list") {
			return deny(req.ID, "permission_denied", "file list is disabled by policy")
		}
		var in struct {
			Limit int `json:"limit"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		entries, err := fs.List(req.Resource, in.Limit)
		if err != nil {
			return deny(req.ID, "permission_denied", err.Error())
		}
		return ok(req.ID, map[string]any{"path": req.Resource, "entries": entries})
	case "file.stat":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("stat") {
			return deny(req.ID, "permission_denied", "file stat is disabled by policy")
		}
		st, err := fs.Stat(req.Resource, false)
		if err != nil {
			return deny(req.ID, "permission_denied", err.Error())
		}
		return ok(req.ID, st)
	case "file.hash":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("hash") {
			return deny(req.ID, "permission_denied", "file hash is disabled by policy")
		}
		sum, err := fs.Hash(req.Resource)
		if err != nil {
			return deny(req.ID, "permission_denied", err.Error())
		}
		return ok(req.ID, map[string]any{"path": req.Resource, "sha256": sum})
	case "file.mkdir":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("mkdir") {
			return deny(req.ID, "permission_denied", "mkdir is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "mkdir requires invocation id")
		}
		return b.mkdir(ctx, req, fs)
	case "file.write", "file.write_test":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("write") {
			return deny(req.ID, "permission_denied", "file write is disabled by policy")
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
		return b.writeFile(ctx, req, []byte(in.Content), fs)
	case "file.patch":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("patch") {
			return deny(req.ID, "permission_denied", "file patch is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "patch requires invocation id")
		}
		var in struct {
			OldText        string `json:"old_text"`
			NewText        string `json:"new_text"`
			ExpectedSHA256 string `json:"expected_sha256"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
		return b.fileMutation(ctx, req, in, func() (any, error) {
			sum, err := fs.Patch(req.Resource, in.OldText, in.NewText, in.ExpectedSHA256)
			return map[string]any{"path": req.Resource, "sha256": sum, "patched": err == nil}, err
		})
	case "file.copy":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("copy") {
			return deny(req.ID, "permission_denied", "file copy is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "copy requires invocation id")
		}
		var in struct {
			Destination string `json:"destination"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil || in.Destination == "" {
			return deny(req.ID, "invalid_args", "destination is required")
		}
		return b.fileMutation(ctx, req, in, func() (any, error) {
			err := fs.CopyFile(req.Resource, in.Destination)
			return map[string]any{"source": req.Resource, "destination": in.Destination, "copied": err == nil}, err
		})
	case "file.move":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("move") {
			return deny(req.ID, "permission_denied", "file move is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "move requires invocation id")
		}
		var in struct {
			Destination string `json:"destination"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil || in.Destination == "" {
			return deny(req.ID, "invalid_args", "destination is required")
		}
		return b.fileMutation(ctx, req, in, func() (any, error) {
			err := fs.Move(req.Resource, in.Destination)
			return map[string]any{"source": req.Resource, "destination": in.Destination, "moved": err == nil}, err
		})
	case "file.remove":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "remove requires invocation id")
		}
		var in struct {
			Recursive bool `json:"recursive"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		action := "remove"
		if in.Recursive {
			action = "remove_recursive"
		}
		if !b.Policy.CanFilesystem(action) {
			return deny(req.ID, "permission_denied", action+" is disabled by policy")
		}
		return b.fileMutation(ctx, req, in, func() (any, error) {
			err := fs.Remove(req.Resource, in.Recursive)
			return map[string]any{"path": req.Resource, "removed": err == nil, "recursive": in.Recursive}, err
		})
	case "file.chmod":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("chmod") {
			return deny(req.ID, "permission_denied", "chmod is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "chmod requires invocation id")
		}
		var in struct {
			Mode uint32 `json:"mode"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil || in.Mode > 0o777 {
			return deny(req.ID, "invalid_args", "mode must be an integer between 0 and 0777")
		}
		return b.fileMutation(ctx, req, in, func() (any, error) {
			err := fs.Chmod(req.Resource, os.FileMode(in.Mode))
			return map[string]any{"path": req.Resource, "mode": in.Mode, "changed": err == nil}, err
		})
	case "file.chown":
		fs, fsErr := b.effectiveFS(ctx, req.Subject)
		if fsErr != nil {
			return deny(req.ID, "filesystem_unavailable", fsErr.Error())
		}
		if !b.Policy.CanFilesystem("chown") {
			return deny(req.ID, "permission_denied", "chown is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "chown requires invocation id")
		}
		var in struct {
			UID int `json:"uid"`
			GID int `json:"gid"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
		return b.fileMutation(ctx, req, in, func() (any, error) {
			err := fs.Chown(req.Resource, in.UID, in.GID)
			return map[string]any{"path": req.Resource, "uid": in.UID, "gid": in.GID, "changed": err == nil}, err
		})
	case "service.list":
		if b.Services == nil {
			return deny(req.ID, "service_unavailable", "service manager is not configured")
		}
		all, err := b.Services.List(ctx)
		if err != nil {
			return deny(req.ID, "service_error", err.Error())
		}
		filtered := make([]ServiceInfo, 0, len(all))
		for _, svc := range all {
			if b.Policy.CanService(svc.Name, "status") || b.Policy.CanService(svc.Name, "inspect") {
				filtered = append(filtered, svc)
			}
		}
		return ok(req.ID, map[string]any{"services": filtered})
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
	case "service.logs":
		if b.Services == nil {
			return deny(req.ID, "service_unavailable", "service manager is not configured")
		}
		if !b.Policy.CanService(req.Resource, "logs") {
			return deny(req.ID, "permission_denied", "service logs are outside policy")
		}
		var in struct {
			Lines int `json:"lines"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		logs, err := b.Services.Logs(ctx, req.Resource, in.Lines)
		if err != nil {
			return deny(req.ID, "service_error", err.Error())
		}
		return ok(req.ID, map[string]any{"service": req.Resource, "logs": logs})
	case "service.start", "service.stop", "service.restart", "service.reload", "service.enable", "service.disable":
		if b.Services == nil {
			return deny(req.ID, "service_unavailable", "service manager is not configured")
		}
		action := strings.TrimPrefix(req.Tool, "service.")
		if !b.Policy.CanService(req.Resource, action) {
			return deny(req.ID, "permission_denied", "service action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "service mutation requires invocation id")
		}
		return b.serviceMutation(ctx, req, action)
	case "docker.list":
		if b.Docker == nil {
			return deny(req.ID, "docker_unavailable", "docker manager is not configured")
		}
		all, err := b.Docker.List(ctx)
		if err != nil {
			return deny(req.ID, "docker_error", err.Error())
		}
		filtered := make([]map[string]any, 0, len(all))
		for _, row := range all {
			name, _ := row["Names"].(string)
			if name == "" {
				name, _ = row["Name"].(string)
			}
			if b.Policy.CanDocker(name, "inspect") || b.Policy.CanDocker(name, "list") {
				filtered = append(filtered, row)
			}
		}
		return ok(req.ID, map[string]any{"resources": filtered})
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
		switch in.Action {
		case "start", "stop", "restart":
		default:
			return deny(req.ID, "unsupported_action", "supported Docker actions are start, stop, restart")
		}
		if !b.Policy.CanDocker(req.Resource, in.Action) {
			return deny(req.ID, "permission_denied", "docker action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "docker action requires invocation id")
		}
		return b.dockerMutation(ctx, req, in.Action)
	case "compose.validate":
		if b.Docker == nil {
			return deny(req.ID, "docker_unavailable", "docker manager is not configured")
		}
		composeOK, composeErr := b.canCompose(ctx, req.Subject, req.Resource, "validate")
		if composeErr != nil {
			return deny(req.ID, "state_error", composeErr.Error())
		}
		if !composeOK {
			return deny(req.ID, "permission_denied", "compose project is outside policy")
		}
		if err := b.Docker.ComposeValidate(ctx, req.Resource); err != nil {
			return deny(req.ID, "compose_error", err.Error())
		}
		return ok(req.ID, map[string]any{"project_dir": req.Resource, "valid": true})
	case "compose.pull", "compose.up", "compose.down":
		if b.Docker == nil {
			return deny(req.ID, "docker_unavailable", "docker manager is not configured")
		}
		action := strings.TrimPrefix(req.Tool, "compose.")
		composeOK, composeErr := b.canCompose(ctx, req.Subject, req.Resource, action)
		if composeErr != nil {
			return deny(req.ID, "state_error", composeErr.Error())
		}
		if !composeOK {
			return deny(req.ID, "permission_denied", "compose action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "compose mutation requires invocation id")
		}
		return b.composeMutation(ctx, req, action)
	case "shell.exec", "job.start":
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "job start requires invocation id")
		}
		return b.startShellJob(ctx, req, false)
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
		max := b.Policy.Shell.MaxOutputBytes
		if max <= 0 {
			max = 1 << 20
		}
		truncated := false
		if len(out) > max {
			out = out[len(out)-max:]
			truncated = true
		}
		return ok(req.ID, map[string]any{"job_id": req.Resource, "output": out, "truncated": truncated})
	case "job.cancel":
		if b.Jobs == nil {
			return deny(req.ID, "jobs_unavailable", "job manager is not configured")
		}
		if err := b.Jobs.Cancel(ctx, req.Subject, req.Resource); err != nil {
			return deny(req.ID, "job_error", err.Error())
		}
		return ok(req.ID, map[string]any{"job_id": req.Resource, "cancelled": true})
	case "package.list":
		var in struct { Limit int `json:"limit"` }
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		rows, err := packageList(ctx, in.Limit)
		if err != nil {
			return deny(req.ID, "package_error", err.Error())
		}
		filtered := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			name, _ := row["name"].(string)
			if b.Policy.CanPackage(name, "list") {
				filtered = append(filtered, row)
			}
		}
		return ok(req.ID, map[string]any{"packages": filtered})
	case "package.update", "package.install", "package.remove":
		action := strings.TrimPrefix(req.Tool, "package.")
		if !b.Policy.CanPackage(req.Resource, action) {
			return deny(req.ID, "permission_denied", "package action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "package action requires invocation id")
		}
		return b.adminMutation(ctx, req, "apt", func() (any, error) {
			out, err := aptAction(ctx, action, req.Resource)
			return map[string]any{"action": action, "package": req.Resource, "output": out}, err
		})
	case "user.list":
		rows, err := userList(ctx)
		if err != nil {
			return deny(req.ID, "identity_error", err.Error())
		}
		filtered := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			name, _ := row["name"].(string)
			if b.Policy.CanUser(name, "list") {
				filtered = append(filtered, row)
			}
		}
		return ok(req.ID, map[string]any{"users": filtered})
	case "user.inspect":
		if !b.Policy.CanUser(req.Resource, "inspect") {
			return deny(req.ID, "permission_denied", "user inspection is outside policy")
		}
		row, err := userInspect(ctx, req.Resource)
		if err != nil {
			return deny(req.ID, "identity_error", err.Error())
		}
		return ok(req.ID, row)
	case "user.add", "user.delete", "user.lock", "user.unlock":
		action := strings.TrimPrefix(req.Tool, "user.")
		if !b.Policy.CanUser(req.Resource, action) {
			return deny(req.ID, "permission_denied", "user action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "user action requires invocation id")
		}
		var in struct { CreateHome bool `json:"create_home"` }
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		return b.adminMutation(ctx, req, "identity:user:"+req.Resource, func() (any, error) {
			out, err := userAction(ctx, action, req.Resource, in.CreateHome)
			return map[string]any{"action": action, "user": req.Resource, "output": out}, err
		})
	case "group.list":
		rows, err := groupList(ctx)
		if err != nil {
			return deny(req.ID, "identity_error", err.Error())
		}
		filtered := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			name, _ := row["name"].(string)
			if b.Policy.CanGroup(name, "list") {
				filtered = append(filtered, row)
			}
		}
		return ok(req.ID, map[string]any{"groups": filtered})
	case "group.inspect":
		if !b.Policy.CanGroup(req.Resource, "inspect") {
			return deny(req.ID, "permission_denied", "group inspection is outside policy")
		}
		row, err := groupInspect(ctx, req.Resource)
		if err != nil {
			return deny(req.ID, "identity_error", err.Error())
		}
		return ok(req.ID, row)
	case "group.add", "group.delete":
		action := strings.TrimPrefix(req.Tool, "group.")
		if !b.Policy.CanGroup(req.Resource, action) {
			return deny(req.ID, "permission_denied", "group action is outside policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "group action requires invocation id")
		}
		return b.adminMutation(ctx, req, "identity:group:"+req.Resource, func() (any, error) {
			out, err := groupAction(ctx, action, req.Resource)
			return map[string]any{"action": action, "group": req.Resource, "output": out}, err
		})
	case "firewall.status":
		if !b.Policy.CanFirewall("status") {
			return deny(req.ID, "permission_denied", "firewall status is disabled by policy")
		}
		out, err := firewallStatus(ctx)
		if err != nil {
			return deny(req.ID, "firewall_error", err.Error())
		}
		return ok(req.ID, map[string]any{"status": out})
	case "firewall.action":
		var in struct {
			Action   string `json:"action"`
			Port     string `json:"port"`
			Protocol string `json:"protocol"`
			Source   string `json:"source"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
		if !b.Policy.CanFirewall(in.Action) {
			return deny(req.ID, "permission_denied", "firewall action is disabled by policy")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "firewall action requires invocation id")
		}
		return b.adminMutation(ctx, req, "firewall", func() (any, error) {
			out, err := firewallAction(ctx, in.Action, in.Port, in.Protocol, in.Source)
			return map[string]any{"action": in.Action, "port": in.Port, "protocol": in.Protocol, "source": in.Source, "output": out}, err
		})
	case "permissions.request_root_access":
		return b.requestRootAccess(ctx, req)
	case "permissions.revoke_root_access":
		return b.revokeRootAccess(ctx, req)
	case "permissions.list_root_access":
		if b.State == nil {
			return deny(req.ID, "state_required", "root delegation listing requires durable state")
		}
		delegations, err := b.State.ListActiveRootDelegations(ctx, req.Subject)
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		return ok(req.ID, map[string]any{
			"physical_scope_root": os.Getenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT"),
			"static": map[string]any{
				"filesystem_read": b.Policy.Filesystem.Read,
				"filesystem_write": b.Policy.Filesystem.Write,
				"shell_cwd_roots": b.Policy.Shell.CWDRoots,
				"compose_inspect": b.Policy.Compose.Inspect,
				"compose_manage": b.Policy.Compose.Manage,
			},
			"dynamic": delegations,
		})
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
		b.elevationMu.Lock()
		defer b.elevationMu.Unlock()
		return b.decideApproval(ctx, req, "approved")
	case "admin.approval.deny":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		b.elevationMu.Lock()
		defer b.elevationMu.Unlock()
		return b.decideApproval(ctx, req, "denied")
	case "admin.audit.tail":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		var in struct {
			AfterSeq int64 `json:"after_seq"`
			Limit    int   `json:"limit"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &in); err != nil {
				return deny(req.ID, "invalid_args", err.Error())
			}
		}
		records, err := b.State.ListAuditAfter(ctx, in.AfterSeq, in.Limit)
		if err != nil {
			return deny(req.ID, "audit_error", err.Error())
		}
		return ok(req.ID, map[string]any{"events": records})
	case "admin.health":
		if !b.adminOK(req.AdminToken) {
			return deny(req.ID, "permission_denied", "operator authentication failed")
		}
		return ok(req.ID, b.healthSnapshot(ctx))
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
		b.elevationMu.Lock()
		defer b.elevationMu.Unlock()
		cancelled, err := b.revokeAllElevatedAccess(ctx)
		if err != nil {
			return deny(req.ID, "revoke_incomplete", err.Error())
		}
		return ok(req.ID, map[string]any{"revoked": true, "cancelled_elevated_jobs": cancelled})
	case "shell.exec_admin":
		if !b.Policy.Features.FullModeEnabled || !b.Policy.Enabled {
			return deny(req.ID, "full_disabled", "full mode is disabled")
		}
		if req.InvocationID == "" {
			return deny(req.ID, "invocation_required", "admin shell requires invocation id")
		}
		b.elevationMu.Lock()
		defer b.elevationMu.Unlock()
		valid, err := b.State.ValidateGrant(ctx, req.GrantID, req.Subject, "shell.admin")
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		if !valid {
			return deny(req.ID, "permission_denied", "valid shell.admin grant required")
		}
		return b.startShellJob(ctx, req, true)
	default:
		return deny(req.ID, "unknown_tool", "unknown broker tool")
	}
}

func (b *Broker) healthSnapshot(ctx context.Context) map[string]any {
	auditOK := true
	activeJobs := 0
	if b.State != nil {
		auditOK = b.State.VerifyAudit(ctx) == nil
		if jobs, err := b.State.ListActiveJobs(ctx); err == nil {
			activeJobs = len(jobs)
		}
	}
	return map[string]any{
		"ok":               true,
		"audit_chain_ok":   auditOK,
		"state_configured": b.State != nil,
		"docker_configured": b.Docker != nil,
		"jobs_configured":  b.Jobs != nil,
		"active_jobs":      activeJobs,
	}
}

func (b *Broker) adminMutation(ctx context.Context, req wire.Request, lockResource string, fn func() (any, error)) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "administrative mutation requires durable state")
	}
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "resource": req.Resource, "args": json.RawMessage(req.Args),
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
	decision, cached, err := b.State.BeginOperation(ctx, req.InvocationID, req.Subject, req.Tool, requestHash)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	switch decision {
	case state.OperationCached:
		return wire.Response{ID: req.ID, OK: true, Result: cached}
	case state.OperationConflict:
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different administrative request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous administrative action outcome is uncertain; inspect state before retry")
	}
	lock, err := b.State.AcquireLock(ctx, lockResource, req.InvocationID, 30*time.Minute)
	if err != nil {
		return deny(req.ID, "resource_busy", err.Error())
	}
	defer b.State.ReleaseLock(context.Background(), lock)

	value, err := fn()
	if err != nil {
		// Deliberately keep the journal pending: root-level commands may have
		// partially changed the host even when their process exits non-zero.
		return deny(req.ID, "admin_action_error", err.Error())
	}
	result, err := json.Marshal(value)
	if err != nil {
		return deny(req.ID, "encode_error", err.Error())
	}
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "administrative action completed but journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

type shellStartArgs struct {
	Command        string `json:"command"`
	CWD            string `json:"cwd"`
	RuntimeSeconds int    `json:"runtime_seconds,omitempty"`
	MemoryBytes    int64  `json:"memory_bytes,omitempty"`
	TasksMax       int    `json:"tasks_max,omitempty"`
}

func shellUnit(invocationID string) string {
	sum := sha256.Sum256([]byte(invocationID))
	return "vps-agent-job-" + hex.EncodeToString(sum[:8])
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func scopedInaccessiblePaths() []string {
	return []string{"/boot", "/etc", "/home", "/media", "/mnt", "/opt", "/root", "/run", "/srv", "/var"}
}

func (b *Broker) startShellJob(ctx context.Context, req wire.Request, admin bool) wire.Response {
	if b.Jobs == nil || b.State == nil {
		return deny(req.ID, "jobs_unavailable", "job manager is not configured")
	}
	var in shellStartArgs
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	if strings.TrimSpace(in.Command) == "" || in.CWD == "" {
		return deny(req.ID, "invalid_args", "command and cwd are required")
	}
	shellOK, shellErr := b.canShellCWD(ctx, req.Subject, in.CWD)
	if shellErr != nil {
		return deny(req.ID, "state_error", shellErr.Error())
	}
	if !shellOK {
		return deny(req.ID, "permission_denied", "shell cwd is outside policy or shell is disabled")
	}

	runtimeLimit := b.Policy.ShellRuntimeLimit()
	runtimeRequested := time.Duration(in.RuntimeSeconds) * time.Second
	if runtimeRequested <= 0 {
		runtimeRequested = runtimeLimit
	}
	if runtimeRequested > runtimeLimit {
		return deny(req.ID, "permission_denied", "requested runtime exceeds shell policy")
	}
	if !b.Policy.CanShellCWD(in.CWD) {
		delegations, err := b.activeRootDelegations(ctx, req.Subject)
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		for _, d := range delegations {
			if (d.Access == "work" || d.Access == "compose") && pathWithinRoot(d.Root, in.CWD) && d.ExpiresAt != nil {
				remaining := time.Until(*d.ExpiresAt)
				if remaining <= 0 {
					return deny(req.ID, "permission_denied", "root delegation has expired")
				}
				if runtimeRequested > remaining {
					runtimeRequested = remaining
				}
				break
			}
		}
	}
	memory := in.MemoryBytes
	if memory <= 0 {
		memory = b.Policy.ShellMemoryLimit()
	}
	if memory > b.Policy.ShellMemoryLimit() {
		return deny(req.ID, "permission_denied", "requested memory exceeds shell policy")
	}
	tasks := in.TasksMax
	if tasks <= 0 {
		tasks = b.Policy.ShellTasksLimit()
	}
	if tasks > b.Policy.ShellTasksLimit() {
		return deny(req.ID, "permission_denied", "requested task limit exceeds shell policy")
	}

	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "args": in, "admin": admin, "grant_id": req.GrantID,
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
	decision, cached, err := b.State.BeginOperation(ctx, req.InvocationID, req.Subject, req.Tool, requestHash)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	switch decision {
	case state.OperationCached:
		return wire.Response{ID: req.ID, OK: true, Result: cached}
	case state.OperationConflict:
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different job request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous job start outcome is uncertain; inspect job state")
	}

	readOnly, readWrite, rootsErr := b.effectiveFileRoots(ctx, req.Subject)
	if rootsErr != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "state_error", rootsErr.Error())
	}
	inaccessible := []string(nil)
	if !b.Policy.ShellMayReadHost() {
		inaccessible = scopedInaccessiblePaths()
	}

	networkMode := b.Policy.Network.Mode
	if networkMode == "" {
		networkMode = "blocked"
	}
	user := b.Policy.Shell.RunAs
	if user == "" {
		user = "vps-agent-exec"
	}
	if admin {
		user = "root"
	}

	spec := sandbox.Spec{
		Unit: shellUnit(req.InvocationID), User: user, Command: in.Command, CWD: in.CWD,
		ReadOnlyPaths: uniqueStrings(readOnly), ReadWritePaths: uniqueStrings(readWrite),
		InaccessiblePaths: inaccessible, IsolateFilesystem: !admin && !b.Policy.ShellMayReadHost(),
		Runtime: runtimeRequested, MemoryMaxBytes: memory, TasksMax: tasks,
		NetworkMode: networkMode, Admin: admin,
	}
	rec, err := b.Jobs.Start(ctx, req.Subject, req.Tool, in.CWD, req.GrantID, spec)
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "job_error", err.Error())
	}
	result, err := json.Marshal(rec)
	if err != nil {
		return deny(req.ID, "encode_error", err.Error())
	}
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "job started but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func pathWithinRoot(root, target string) bool {
	if !filepath.IsAbs(root) || !filepath.IsAbs(target) {
		return false
	}
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func (b *Broker) activeRootDelegations(ctx context.Context, subject string) ([]state.RootDelegation, error) {
	if b.State == nil || subject == "" {
		return nil, nil
	}
	return b.State.ListActiveRootDelegations(ctx, subject)
}

func (b *Broker) effectiveFileRoots(ctx context.Context, subject string) ([]string, []string, error) {
	if b.Policy == nil {
		return nil, nil, errors.New("policy is not configured")
	}
	readRoots := append([]string(nil), b.Policy.Filesystem.Read...)
	writeRoots := append([]string(nil), b.Policy.Filesystem.Write...)
	delegations, err := b.activeRootDelegations(ctx, subject)
	if err != nil {
		return nil, nil, err
	}
	for _, d := range delegations {
		switch d.Access {
		case "read":
			readRoots = append(readRoots, d.Root)
		case "work", "compose":
			readRoots = append(readRoots, d.Root)
			writeRoots = append(writeRoots, d.Root)
		}
	}
	return uniqueStrings(readRoots), uniqueStrings(writeRoots), nil
}

func (b *Broker) effectiveFS(ctx context.Context, subject string) (*securefs.Manager, error) {
	readRoots, writeRoots, err := b.effectiveFileRoots(ctx, subject)
	if err != nil {
		return nil, err
	}
	return securefs.NewWithHostRoot(
		readRoots,
		writeRoots,
		securefs.DefaultMaxBytes,
		os.Getenv("VPS_AGENT_HOST_ROOT"),
	)
}

func (b *Broker) canShellCWD(ctx context.Context, subject, cwd string) (bool, error) {
	if b.Policy == nil || !b.Policy.Shell.Enabled {
		return false, nil
	}
	if b.Policy.CanShellCWD(cwd) {
		return true, nil
	}
	delegations, err := b.activeRootDelegations(ctx, subject)
	if err != nil {
		return false, err
	}
	for _, d := range delegations {
		if (d.Access == "work" || d.Access == "compose") && pathWithinRoot(d.Root, cwd) {
			return true, nil
		}
	}
	return false, nil
}

func (b *Broker) canCompose(ctx context.Context, subject, projectDir, action string) (bool, error) {
	if b.Policy == nil || !b.Policy.CanComposeAction(action) {
		return false, nil
	}
	if b.Policy.CanCompose(projectDir, action) {
		return true, nil
	}
	delegations, err := b.activeRootDelegations(ctx, subject)
	if err != nil {
		return false, err
	}
	projectDir = filepath.Clean(projectDir)
	for _, d := range delegations {
		if d.Access == "compose" && filepath.Clean(d.Root) == projectDir {
			return true, nil
		}
	}
	return false, nil
}

func (b *Broker) normalizeDelegatedRoot(root, access string) (string, string, error) {
	if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) {
		return "", "", errors.New("root must be an absolute path")
	}
	root = filepath.Clean(root)
	access = strings.ToLower(strings.TrimSpace(access))
	switch access {
	case "read", "work", "compose":
	default:
		return "", "", errors.New("access must be read, work, or compose")
	}
	physical := os.Getenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT")
	if physical == "" || !filepath.IsAbs(physical) {
		return "", "", errors.New("physical scope root is not configured")
	}
	physical = filepath.Clean(physical)
	if !pathWithinRoot(physical, root) {
		return "", "", fmt.Errorf("root %q is outside physical scope %q", root, physical)
	}
	if root == physical {
		return "", "", errors.New("dynamic delegation of the entire physical ceiling is not allowed")
	}
	if _, err := securefs.NewWithHostRoot(
		[]string{root},
		[]string{root},
		securefs.DefaultMaxBytes,
		os.Getenv("VPS_AGENT_HOST_ROOT"),
	); err != nil {
		return "", "", fmt.Errorf("unsafe delegated root: %w", err)
	}
	return root, access, nil
}

func (b *Broker) requestRootAccess(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "root delegation requires durable state")
	}
	if req.InvocationID == "" {
		return deny(req.ID, "invocation_required", "root delegation request requires invocation id")
	}
	var in struct {
		Root       string `json:"root"`
		Access     string `json:"access"`
		TTLSeconds int64  `json:"ttl_seconds"`
	}
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	root, access, err := b.normalizeDelegatedRoot(in.Root, in.Access)
	if err != nil {
		return deny(req.ID, "permission_denied", err.Error())
	}
	if in.TTLSeconds < 0 {
		return deny(req.ID, "invalid_ttl", "ttl_seconds cannot be negative")
	}
	ttl := time.Duration(in.TTLSeconds) * time.Second
	if ttl > 0 && ttl > b.Policy.MaxGrantTTL() {
		return deny(req.ID, "invalid_ttl", "temporary delegation ttl exceeds policy")
	}
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "root": root, "access": access, "ttl_seconds": in.TTLSeconds,
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
	decision, cached, err := b.State.BeginOperation(ctx, req.InvocationID, req.Subject, req.Tool, requestHash)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	switch decision {
	case state.OperationCached:
		return wire.Response{ID: req.ID, OK: true, Result: cached}
	case state.OperationConflict:
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different root delegation request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous root delegation request outcome is uncertain")
	}
	a, err := b.State.CreateRootApproval(ctx, req.Subject, root, access, ttl, 10*time.Minute)
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "state_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{
		"request_id": a.ID,
		"status": a.Status,
		"root": root,
		"access": access,
		"permanent": ttl == 0,
		"delegation_ttl_seconds": in.TTLSeconds,
		"approval_expires_at": a.ExpiresAt,
		"approval_required": true,
	})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "approval request created but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func (b *Broker) revokeRootAccess(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "root delegation revocation requires durable state")
	}
	if req.InvocationID == "" {
		return deny(req.ID, "invocation_required", "root delegation revocation requires invocation id")
	}
	var in struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	if strings.TrimSpace(in.Root) == "" || !filepath.IsAbs(in.Root) {
		return deny(req.ID, "invalid_args", "root must be an absolute path")
	}
	root := filepath.Clean(in.Root)
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "root": root,
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
	decision, cached, err := b.State.BeginOperation(ctx, req.InvocationID, req.Subject, req.Tool, requestHash)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	switch decision {
	case state.OperationCached:
		return wire.Response{ID: req.ID, OK: true, Result: cached}
	case state.OperationConflict:
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different revocation request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous root delegation revocation outcome is uncertain")
	}
	count, err := b.State.RevokeRootDelegation(ctx, req.Subject, root)
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "state_error", err.Error())
	}
	cancelledJobs := 0
	if count > 0 && b.Jobs != nil {
		active, listErr := b.State.ListActiveJobs(ctx)
		if listErr != nil {
			return deny(req.ID, "revoke_incomplete", "delegation revoked but active jobs could not be listed: "+listErr.Error())
		}
		for _, rec := range active {
			if rec.Subject != req.Subject || !pathWithinRoot(root, rec.Resource) || b.Policy.CanShellCWD(rec.Resource) {
				continue
			}
			if err := b.Jobs.Cancel(ctx, rec.Subject, rec.ID); err != nil {
				return deny(req.ID, "revoke_incomplete", "delegation revoked but dependent job cancellation failed: "+err.Error())
			}
			cancelledJobs++
		}
	}
	result, _ := json.Marshal(map[string]any{
		"root": root,
		"revoked": count > 0,
		"revoked_dynamic_delegations": count,
		"cancelled_dependent_jobs": cancelledJobs,
	})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "delegation revoked but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
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
	pending, err := b.State.GetApproval(ctx, in.RequestID)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	if decision == "approved" && pending.Kind == "root" {
		root, access, err := b.normalizeDelegatedRoot(pending.Resource, pending.Access)
		if err != nil {
			return deny(req.ID, "permission_denied", "root approval no longer satisfies the physical security boundary: "+err.Error())
		}
		pending.Resource = root
		pending.Access = access
	}
	a, err := b.State.DecideApproval(ctx, in.RequestID, decision)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	if decision == "denied" {
		return ok(req.ID, map[string]any{"request_id": a.ID, "status": "denied", "kind": a.Kind})
	}
	if a.Kind == "root" {
		d, err := b.State.IssueRootDelegation(ctx, a.Subject, a.Resource, a.Access, a.ID, a.TTL)
		if err != nil {
			return deny(req.ID, "state_error", err.Error())
		}
		return ok(req.ID, map[string]any{
			"request_id": a.ID,
			"status": "approved",
			"kind": "root",
			"delegation": d,
		})
	}
	g, err := b.State.IssueGrant(ctx, a.Subject, a.Capabilities, a.TTL)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	return ok(req.ID, map[string]any{
		"request_id": a.ID, "status": "approved", "kind": "capability", "grant_id": g.ID,
		"subject": g.Subject, "capabilities": g.Capabilities, "expires_at": g.ExpiresAt,
	})
}

func (b *Broker) revokeAllElevatedAccess(ctx context.Context) (int, error) {
	if b.State == nil {
		return 0, errors.New("durable state is required")
	}
	if err := b.State.RevokeAll(ctx); err != nil {
		return 0, err
	}

	active, err := b.State.ListActiveJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("grants revoked but active jobs could not be listed: %w", err)
	}
	if len(active) == 0 {
		return 0, nil
	}
	if b.Jobs == nil {
		for _, rec := range active {
			if rec.GrantID != "" {
				return 0, errors.New("grants revoked but elevated jobs cannot be cancelled because the job manager is unavailable")
			}
		}
		return 0, nil
	}

	cancelled := 0
	failed := make([]string, 0)
	for _, rec := range active {
		dependedOnElevation := rec.GrantID != ""
		dependedOnDynamicRoot := rec.Resource != "" && !b.Policy.CanShellCWD(rec.Resource)
		if !dependedOnElevation && !dependedOnDynamicRoot {
			continue
		}
		if err := b.Jobs.Cancel(ctx, rec.Subject, rec.ID); err != nil {
			failed = append(failed, rec.ID+": "+err.Error())
			continue
		}
		cancelled++
	}
	if len(failed) > 0 {
		return cancelled, fmt.Errorf("grants revoked, but %d elevated job(s) could not be cancelled: %s", len(failed), strings.Join(failed, "; "))
	}
	return cancelled, nil
}

func (b *Broker) adminOK(token string) bool {
	if b.AdminToken == "" || token == "" || len(token) != len(b.AdminToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(b.AdminToken)) == 1
}



func (b *Broker) fileMutation(ctx context.Context, req wire.Request, fingerprint any, fn func() (any, error)) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "filesystem mutation requires durable state")
	}
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "resource": req.Resource, "args": fingerprint,
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
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
		return deny(req.ID, "reconcile_required", "previous operation outcome is uncertain")
	}
	value, err := fn()
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "filesystem_error", err.Error())
	}
	result, err := json.Marshal(value)
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "encode_error", err.Error())
	}
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "filesystem operation completed but journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func (b *Broker) mkdir(ctx context.Context, req wire.Request, fs *securefs.Manager) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "mkdir requires durable state")
	}
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "resource": req.Resource,
	})
	if err != nil {
		return deny(req.ID, "hash_error", err.Error())
	}
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
		return deny(req.ID, "reconcile_required", "previous mkdir outcome is uncertain")
	}

	if err := fs.MkdirAll(req.Resource, 0o750); err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "permission_denied", err.Error())
	}
	result, _ := json.Marshal(map[string]any{
		"path": req.Resource, "created": true,
	})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "directory created but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func (b *Broker) writeFile(ctx context.Context, req wire.Request, data []byte, fs *securefs.Manager) wire.Response {
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
	if err := fs.WriteFileAtomic(req.Resource, data); err != nil {
		if b.State != nil {
			_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		}
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

func (b *Broker) serviceMutation(ctx context.Context, req wire.Request, action string) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "service mutation requires durable state")
	}
	requestHash, _ := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "service": req.Resource, "action": action,
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
		return deny(req.ID, "reconcile_required", "previous service action outcome is uncertain; inspect service before retry")
	}

	lock, err := b.State.AcquireLock(ctx, "service:"+req.Resource, req.InvocationID, 2*time.Minute)
	if err != nil {
		return deny(req.ID, "resource_busy", err.Error())
	}
	defer b.State.ReleaseLock(context.Background(), lock)

	var status string
	switch action {
	case "start":
		status, err = b.Services.Start(ctx, req.Resource)
	case "stop":
		status, err = b.Services.Stop(ctx, req.Resource)
	case "restart":
		status, err = b.Services.Restart(ctx, req.Resource)
	case "reload":
		status, err = b.Services.Reload(ctx, req.Resource)
	case "enable":
		status, err = b.Services.Enable(ctx, req.Resource)
	case "disable":
		status, err = b.Services.Disable(ctx, req.Resource)
	default:
		err = fmt.Errorf("unsupported service action %q", action)
	}
	if err != nil {
		return deny(req.ID, "service_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"service": req.Resource, "action": action, "status": status})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "service action completed but journal update failed: "+err.Error())
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

func (b *Broker) dockerMutation(ctx context.Context, req wire.Request, action string) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "docker mutation requires durable state")
	}
	requestHash, _ := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "resource": req.Resource, "action": action,
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
		return deny(req.ID, "reconcile_required", "previous Docker action outcome is uncertain; inspect before retry")
	}
	lock, err := b.State.AcquireLock(ctx, "docker:"+req.Resource, req.InvocationID, 5*time.Minute)
	if err != nil {
		return deny(req.ID, "resource_busy", err.Error())
	}
	defer b.State.ReleaseLock(context.Background(), lock)

	var info map[string]any
	switch action {
	case "start":
		info, err = b.Docker.Start(ctx, req.Resource)
	case "stop":
		info, err = b.Docker.Stop(ctx, req.Resource)
	case "restart":
		info, err = b.Docker.Restart(ctx, req.Resource)
	default:
		err = fmt.Errorf("unsupported Docker action %q", action)
	}
	if err != nil {
		return deny(req.ID, "docker_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"resource": req.Resource, "action": action, "inspect": info})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "Docker action completed but journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func (b *Broker) composeMutation(ctx context.Context, req wire.Request, action string) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "compose mutation requires durable state")
	}
	requestHash, _ := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "project_dir": req.Resource, "action": action,
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
		return deny(req.ID, "reconcile_required", "previous Compose action outcome is uncertain; inspect project before retry")
	}
	lock, err := b.State.AcquireLock(ctx, "compose:"+req.Resource, req.InvocationID, 15*time.Minute)
	if err != nil {
		return deny(req.ID, "resource_busy", err.Error())
	}
	defer b.State.ReleaseLock(context.Background(), lock)

	var output string
	switch action {
	case "pull":
		output, err = b.Docker.ComposePull(ctx, req.Resource)
	case "up":
		output, err = b.Docker.ComposeUp(ctx, req.Resource)
	case "down":
		output, err = b.Docker.ComposeDown(ctx, req.Resource)
	default:
		err = fmt.Errorf("unsupported Compose action %q", action)
	}
	if err != nil {
		return deny(req.ID, "compose_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"project_dir": req.Resource, "action": action, "output": output})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "Compose action completed but journal update failed: "+err.Error())
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

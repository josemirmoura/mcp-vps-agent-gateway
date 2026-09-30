package broker

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/hostexec"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sensitive"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

const defaultSensitiveGrantTTL = 15 * time.Minute

func sensitiveCapability(access, path string) string {
	return "sensitive." + access + ":" + hex.EncodeToString([]byte(filepath.Clean(path)))
}

func decodeSensitiveCapability(cap string) (access, path string, ok bool) {
	var prefix string
	switch {
	case strings.HasPrefix(cap, "sensitive.read:"):
		access, prefix = "read", "sensitive.read:"
	case strings.HasPrefix(cap, "sensitive.work:"):
		access, prefix = "work", "sensitive.work:"
	default:
		return "", "", false
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(cap, prefix))
	if err != nil || len(raw) == 0 {
		return "", "", false
	}
	path = filepath.Clean(string(raw))
	if !filepath.IsAbs(path) {
		return "", "", false
	}
	return access, path, true
}

func pathWithinAnyRoot(roots []string, target string) bool {
	for _, root := range roots {
		if pathWithinRoot(root, target) {
			return true
		}
	}
	return false
}

func (b *Broker) activeSensitiveAccess(ctx context.Context, subject string) (map[string]string, error) {
	out := map[string]string{}
	if b.State == nil || subject == "" {
		return out, nil
	}
	grants, err := b.State.ListActiveGrants(ctx, subject)
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		for _, cap := range grant.Capabilities {
			access, path, ok := decodeSensitiveCapability(cap)
			if !ok {
				continue
			}
			if access == "work" || out[path] == "" {
				out[path] = access
			}
		}
	}
	return out, nil
}

func (b *Broker) sensitivePathAllowed(ctx context.Context, subject, target string, write bool) (bool, error) {
	readRoots, writeRoots, err := b.effectiveFileRoots(ctx, subject)
	if err != nil {
		return false, err
	}
	roots := readRoots
	if write {
		roots = writeRoots
	}
	protected, err := sensitive.IsProtected(
		os.Getenv("VPS_AGENT_HOST_ROOT"),
		target,
		roots,
		sensitive.DefaultScanLimit,
	)
	if err != nil {
		return false, fmt.Errorf("sensitive path safety check failed: %w", err)
	}
	if !protected {
		return true, nil
	}
	access, err := b.activeSensitiveAccess(ctx, subject)
	if err != nil {
		return false, err
	}
	granted := access[filepath.Clean(target)]
	if write {
		return granted == "work", nil
	}
	return granted == "read" || granted == "work", nil
}

func (b *Broker) guardSensitiveRequest(ctx context.Context, req wire.Request) *wire.Response {
	type check struct {
		path  string
		write bool
	}
	var checks []check
	switch req.Tool {
	case "file.read", "file.read_test", "file.hash":
		checks = append(checks, check{path: req.Resource})
	case "file.write", "file.write_test", "file.patch", "file.chmod", "file.chown":
		checks = append(checks, check{path: req.Resource, write: true})
	case "file.remove":
		checks = append(checks, check{path: req.Resource, write: true})
		var in struct {
			Recursive bool `json:"recursive"`
		}
		if json.Unmarshal(req.Args, &in) == nil && in.Recursive {
			_, writeRoots, err := b.effectiveFileRoots(ctx, req.Subject)
			if err != nil {
				resp := deny(req.ID, "sensitive_path_check_failed", err.Error())
				return &resp
			}
			if err := b.requireSensitiveSubtreeWork(ctx, req.Subject, req.Resource, writeRoots); err != nil {
				resp := deny(req.ID, "sensitive_path_locked", err.Error())
				return &resp
			}
		}
	case "file.copy", "file.move":
		var in struct {
			Destination string `json:"destination"`
		}
		if json.Unmarshal(req.Args, &in) == nil && in.Destination != "" {
			readRoots, _, err := b.effectiveFileRoots(ctx, req.Subject)
			if err != nil {
				resp := deny(req.ID, "sensitive_path_check_failed", err.Error())
				return &resp
			}
			sourceProtected, err := sensitive.IsProtected(
				os.Getenv("VPS_AGENT_HOST_ROOT"),
				req.Resource,
				readRoots,
				sensitive.DefaultScanLimit,
			)
			if err != nil {
				resp := deny(req.ID, "sensitive_path_check_failed", err.Error())
				return &resp
			}
			if sourceProtected && !sensitive.ProtectedName(in.Destination) {
				resp := deny(req.ID, "sensitive_path_locked",
					"protected secret material cannot be copied or moved to an unprotected filename")
				return &resp
			}
			sourceWrite := req.Tool == "file.move"
			if sourceWrite {
				info, statErr := os.Lstat(hostexec.Path(filepath.Clean(req.Resource)))
				if statErr != nil && !os.IsNotExist(statErr) {
					resp := deny(req.ID, "sensitive_path_check_failed", statErr.Error())
					return &resp
				}
				if statErr == nil && info.IsDir() {
					_, writeRoots, rootsErr := b.effectiveFileRoots(ctx, req.Subject)
					if rootsErr != nil {
						resp := deny(req.ID, "sensitive_path_check_failed", rootsErr.Error())
						return &resp
					}
					if err := b.requireSensitiveSubtreeWork(ctx, req.Subject, req.Resource, writeRoots); err != nil {
						resp := deny(req.ID, "sensitive_path_locked", err.Error())
						return &resp
					}
				}
			}
			checks = append(checks,
				check{path: req.Resource, write: sourceWrite},
				check{path: in.Destination, write: true},
			)
		}
	default:
		return nil
	}
	for _, item := range checks {
		if item.path == "" {
			continue
		}
		allowed, err := b.sensitivePathAllowed(ctx, req.Subject, item.path, item.write)
		if err != nil {
			resp := deny(req.ID, "sensitive_path_check_failed", err.Error())
			return &resp
		}
		if allowed {
			continue
		}
		resp := deny(req.ID, "sensitive_path_locked",
			"protected secret path requires a separate explicit human approval")
		return &resp
	}
	return nil
}

func (b *Broker) sensitiveShellMasks(ctx context.Context, subject string, roots []string) ([]string, error) {
	aliases, err := sensitive.ProtectedAliases(
		os.Getenv("VPS_AGENT_HOST_ROOT"),
		roots,
		sensitive.DefaultScanLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot safely prepare shell secret mask: %w", err)
	}
	access, err := b.activeSensitiveAccess(ctx, subject)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(aliases))
	for _, path := range aliases {
		// Arbitrary shell commands can write as well as read. Only a separate
		// sensitive "work" approval exposes a protected path to shell.exec.
		if access[filepath.Clean(path)] == "work" {
			continue
		}
		out = append(out, filepath.Clean(path))
	}
	return out, nil
}

func (b *Broker) discoverScope(ctx context.Context, req wire.Request) wire.Response {
	physical := filepath.Clean(os.Getenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT"))
	if physical == "." || physical == "" || !filepath.IsAbs(physical) {
		return deny(req.ID, "scope_unavailable", "physical filesystem ceiling is not configured")
	}
	var in struct {
		Limit int `json:"limit"`
	}
	if len(req.Args) > 0 {
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return deny(req.ID, "invalid_args", err.Error())
		}
	}
	if in.Limit <= 0 || in.Limit > 200 {
		in.Limit = 200
	}
	entries, err := os.ReadDir(hostexec.Path(physical))
	if err != nil {
		return deny(req.ID, "scope_unavailable", err.Error())
	}
	readRoots, writeRoots, err := b.effectiveFileRoots(ctx, req.Subject)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	dirs := make([]map[string]any, 0)
	for _, entry := range entries {
		if len(dirs) >= in.Limit {
			break
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		path := filepath.Join(physical, entry.Name())
		dirs = append(dirs, map[string]any{
			"name": entry.Name(),
			"path": path,
			"read_authorized": pathWithinAnyRoot(readRoots, path),
			"work_authorized": pathWithinAnyRoot(writeRoots, path),
		})
	}
	return ok(req.ID, map[string]any{
		"physical_ceiling": physical,
		"discovery_only": true,
		"directories": dirs,
		"note": "Directory names are visible for discovery; contents remain subject to explicit root authorization.",
	})
}

func (b *Broker) sensitiveApprovalToken(a state.Approval) (string, error) {
	if b.AdminToken == "" {
		return "", errors.New("operator approval secret is not configured")
	}
	mac := hmac.New(sha256.New, []byte(b.AdminToken))
	_, _ = fmt.Fprintf(mac, "%s\n%s\n%s\n%d",
		a.ID, a.Subject, strings.Join(a.Capabilities, ","), a.ExpiresAt.UnixNano())
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (b *Broker) validSensitiveApprovalToken(a state.Approval, token string) bool {
	expected, err := b.sensitiveApprovalToken(a)
	if err != nil || token == "" || len(token) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (b *Broker) requestSensitiveAccess(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "sensitive access requires durable state")
	}
	if req.InvocationID == "" {
		return deny(req.ID, "invocation_required", "sensitive access request requires invocation id")
	}
	var in struct {
		Path       string `json:"path"`
		Access     string `json:"access"`
		TTLSeconds int64  `json:"ttl_seconds"`
	}
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	if !filepath.IsAbs(in.Path) {
		return deny(req.ID, "invalid_args", "path must be absolute")
	}
	path := filepath.Clean(in.Path)
	access := strings.ToLower(strings.TrimSpace(in.Access))
	if access != "read" && access != "work" {
		return deny(req.ID, "invalid_args", "access must be read or work")
	}
	readRoots, writeRoots, err := b.effectiveFileRoots(ctx, req.Subject)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	roots := readRoots
	if access == "work" {
		roots = writeRoots
	}
	if !pathWithinAnyRoot(roots, path) {
		return deny(req.ID, "permission_denied", "sensitive path is not inside an already authorized root")
	}
	protected, err := sensitive.IsProtected(os.Getenv("VPS_AGENT_HOST_ROOT"), path, roots, sensitive.DefaultScanLimit)
	if err != nil {
		return deny(req.ID, "sensitive_path_check_failed", err.Error())
	}
	if !protected {
		return deny(req.ID, "invalid_args", "path is not classified as a protected secret")
	}
	if in.TTLSeconds < 0 {
		return deny(req.ID, "invalid_ttl", "ttl_seconds cannot be negative")
	}
	ttl := time.Duration(in.TTLSeconds) * time.Second
	if ttl == 0 {
		ttl = defaultSensitiveGrantTTL
	}
	if ttl > b.Policy.MaxGrantTTL() {
		return deny(req.ID, "invalid_ttl", "temporary sensitive access ttl exceeds policy")
	}
	capability := sensitiveCapability(access, path)
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "path": path, "access": access,
		"ttl_seconds": int64(ttl / time.Second),
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
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different sensitive access request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous sensitive access request outcome is uncertain")
	}
	a, err := b.State.CreateApproval(ctx, req.Subject, []string{capability}, ttl, 10*time.Minute)
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "state_error", err.Error())
	}
	token, err := b.sensitiveApprovalToken(a)
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "approval_unavailable", err.Error())
	}
	result, _ := json.Marshal(map[string]any{
		"request_id": a.ID,
		"status": a.Status,
		"kind": "sensitive",
		"path": path,
		"access": access,
		"delegation_ttl_seconds": int64(ttl / time.Second),
		"approval_expires_at": a.ExpiresAt,
		"approval_required": true,
		"approval_token": token,
	})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "sensitive approval request created but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

func (b *Broker) validateSensitiveApproval(ctx context.Context, approval state.Approval) error {
	if approval.Kind != "capability" || len(approval.Capabilities) != 1 {
		return errors.New("approval is not a protected-file capability")
	}
	access, path, ok := decodeSensitiveCapability(approval.Capabilities[0])
	if !ok {
		return errors.New("approval is not a protected-file capability")
	}
	readRoots, writeRoots, err := b.effectiveFileRoots(ctx, approval.Subject)
	if err != nil {
		return err
	}
	roots := readRoots
	if access == "work" {
		roots = writeRoots
	}
	if !pathWithinAnyRoot(roots, path) {
		return errors.New("parent root authorization is no longer active")
	}
	protected, err := sensitive.IsProtected(
		os.Getenv("VPS_AGENT_HOST_ROOT"),
		path,
		roots,
		sensitive.DefaultScanLimit,
	)
	if err != nil {
		return err
	}
	if !protected {
		return errors.New("path is no longer classified as protected")
	}
	return nil
}

func (b *Broker) confirmSensitiveAccess(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "sensitive access confirmation requires durable state")
	}
	var in struct {
		RequestID     string `json:"request_id"`
		ApprovalToken string `json:"approval_token"`
		Decision      string `json:"decision"`
	}
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	if in.Decision != "approve" && in.Decision != "deny" {
		return deny(req.ID, "invalid_args", "decision must be approve or deny")
	}
	a, err := b.State.GetApproval(ctx, in.RequestID)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	if a.Kind != "capability" || len(a.Capabilities) != 1 {
		return deny(req.ID, "permission_denied", "approval request is not a sensitive-path grant")
	}
	if _, _, ok := decodeSensitiveCapability(a.Capabilities[0]); !ok {
		return deny(req.ID, "permission_denied", "approval request is not a sensitive-path grant")
	}
	if a.Subject != req.Subject {
		return deny(req.ID, "identity_mismatch", "approval request belongs to a different authenticated subject")
	}
	if !b.validSensitiveApprovalToken(a, in.ApprovalToken) {
		return deny(req.ID, "permission_denied", "invalid sensitive approval token")
	}
	decision := "approved"
	if in.Decision == "deny" {
		decision = "denied"
	}
	args, _ := json.Marshal(map[string]any{"request_id": in.RequestID})
	return b.decideApproval(ctx, wire.Request{
		ID: req.ID, Subject: req.Subject, Tool: "permissions.confirm_sensitive_access", Args: args,
	}, decision)
}

func (b *Broker) listSensitiveAccess(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "sensitive access listing requires durable state")
	}
	grants, err := b.State.ListActiveGrants(ctx, req.Subject)
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	rows := make([]map[string]any, 0)
	for _, grant := range grants {
		for _, cap := range grant.Capabilities {
			access, path, ok := decodeSensitiveCapability(cap)
			if !ok {
				continue
			}
			rows = append(rows, map[string]any{
				"grant_id": grant.ID,
				"path": path,
				"access": access,
				"expires_at": grant.ExpiresAt,
			})
		}
	}
	return ok(req.ID, map[string]any{"sensitive": rows})
}

func (b *Broker) revokeSensitiveAccess(ctx context.Context, req wire.Request) wire.Response {
	if b.State == nil {
		return deny(req.ID, "state_required", "sensitive access revocation requires durable state")
	}
	if req.InvocationID == "" {
		return deny(req.ID, "invocation_required", "sensitive access revocation requires invocation id")
	}
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(req.Args, &in); err != nil {
		return deny(req.ID, "invalid_args", err.Error())
	}
	if !filepath.IsAbs(in.Path) {
		return deny(req.ID, "invalid_args", "path must be absolute")
	}
	path := filepath.Clean(in.Path)
	requestHash, err := state.HashRequest(map[string]any{
		"subject": req.Subject, "tool": req.Tool, "path": path,
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
		return deny(req.ID, "idempotency_conflict", "invocation id reused with different sensitive revocation request")
	case state.OperationReconcile:
		return deny(req.ID, "reconcile_required", "previous sensitive revocation outcome is uncertain")
	}
	count, err := b.State.RevokeGrantsByCapability(ctx, req.Subject,
		sensitiveCapability("read", path), sensitiveCapability("work", path))
	if err != nil {
		_ = b.State.AbortOperation(context.Background(), req.InvocationID)
		return deny(req.ID, "state_error", err.Error())
	}
	result, _ := json.Marshal(map[string]any{"path": path, "revoked_grants": count})
	if err := b.State.CompleteOperation(ctx, req.InvocationID, result); err != nil {
		return deny(req.ID, "state_error", "sensitive access revoked but operation journal update failed: "+err.Error())
	}
	return wire.Response{ID: req.ID, OK: true, Result: result}
}

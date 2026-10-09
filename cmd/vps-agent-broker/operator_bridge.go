package main

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

// operatorBridge exposes the minimal approval protocol, not the privileged
// Broker API. Crucially, a web-session compromise cannot approve shell.admin
// or any other capability elevation by submitting a hidden request ID.
type operatorBridge struct {
	broker *broker.Broker
	token  string
}

func eligibleWebApproval(a state.Approval, expectedSubject string) bool {
	if expectedSubject == "" || !utf8.ValidString(a.Subject+a.Resource) || unsafeDisplay(a.Subject+a.Resource) {
		return false
	}
	if a.Status != "pending" || !time.Now().Before(a.ExpiresAt) || a.Subject == "" ||
		a.TTL <= 0 ||
		(expectedSubject != "" && a.Subject != expectedSubject) {
		return false
	}
	if a.Kind == "root" {
		ceiling := os.Getenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT")
		if ceiling != "" && filepath.Clean(a.Resource) == filepath.Clean(ceiling) {
			return false // ceiling-wide grants require interactive operator CLI
		}
		return filepath.IsAbs(a.Resource) &&
			(a.Access == "read" || a.Access == "work" || a.Access == "compose") &&
			len(a.Capabilities) == 0
	}
	if a.Kind != "capability" || len(a.Capabilities) != 1 {
		return false
	}
	cap := a.Capabilities[0]
	var encoded string
	switch {
	case strings.HasPrefix(cap, "sensitive.read:"):
		encoded = strings.TrimPrefix(cap, "sensitive.read:")
	case strings.HasPrefix(cap, "sensitive.work:"):
		encoded = strings.TrimPrefix(cap, "sensitive.work:")
	default:
		return false
	}
	decoded, err := hex.DecodeString(encoded)
	return err == nil && len(decoded) > 0 && filepath.IsAbs(string(decoded)) &&
		!strings.ContainsRune(string(decoded), 0) && utf8.Valid(decoded) && !unsafeDisplay(string(decoded))
}

func (h *operatorBridge) Handle(ctx context.Context, r wire.Request) wire.Response {
	switch r.Tool {
	case "admin.approval.list", "admin.approval.get", "admin.approval.approve", "admin.approval.deny", "admin.approval.status":
	default:
		return wire.ErrorResponse(r.ID, "permission_denied", "operator socket restricts operations")
	}
	if h.broker == nil || h.broker.State == nil || h.token == "" ||
		operatorIdentity() == "" || r.Subject != operatorIdentity() || h.broker.ExpectedSubject == "" || h.broker.InstanceID == "" ||
		len(r.AdminToken) != len(h.token) ||
		subtle.ConstantTimeCompare([]byte(r.AdminToken), []byte(h.token)) != 1 {
		return wire.ErrorResponse(r.ID, "permission_denied", "operator authentication failed")
	}
	// Enforce eligibility on both listing AND decisions, including direct socket
	// calls that bypass the UI. The Broker still performs authoritative checks.
	if r.Tool == "admin.approval.list" {
		all, err := h.broker.State.ListPendingApprovals(ctx)
		if err != nil {
			return wire.ErrorResponse(r.ID, "state_error", "pending approvals unavailable")
		}
		allowed := make([]state.Approval, 0)
		for _, a := range all {
			if eligibleWebApproval(a, h.broker.ExpectedSubject) {
				a.Fingerprint = state.ApprovalFingerprint(a, h.broker.InstanceID)
				a.NodeID = h.broker.InstanceID
				allowed = append(allowed, a)
			}
		}
		raw, err := json.Marshal(allowed)
		if err != nil {
			return wire.ErrorResponse(r.ID, "encode_error", "pending approvals unavailable")
		}
		return wire.Response{ID: r.ID, OK: true, Result: raw}
	}
	var in struct {
		RequestID   string `json:"request_id"`
		Fingerprint string `json:"snapshot_hash"`
		NodeID      string `json:"node_id"`
		StepUp      bool   `json:"step_up"`
	}
	if err := json.Unmarshal(r.Args, &in); err != nil || !strings.HasPrefix(in.RequestID, "apr_") {
		return wire.ErrorResponse(r.ID, "invalid_args", "request ID invalid")
	}
	if r.Tool == "admin.approval.status" {
		r.Subject = h.broker.ExpectedSubject
		r.Tool = "permissions.approval_status"
		return h.broker.Handle(ctx, r)
	}
	a, err := h.broker.State.GetApproval(ctx, in.RequestID)
	if r.Tool == "admin.approval.get" {
		if err != nil || !eligibleWebApproval(a, h.broker.ExpectedSubject) {
			return wire.Response{ID: r.ID, OK: true, Result: json.RawMessage("null")}
		}
		a.Fingerprint = state.ApprovalFingerprint(a, h.broker.InstanceID)
		a.NodeID = h.broker.InstanceID
		raw, err := json.Marshal(a)
		if err != nil {
			return wire.ErrorResponse(r.ID, "encode_error", "request unavailable")
		}
		return wire.Response{ID: r.ID, OK: true, Result: raw}
	}
	if err != nil || !eligibleWebApproval(a, h.broker.ExpectedSubject) {
		return wire.ErrorResponse(r.ID, "permission_denied", "request not eligible for web approval")
	}
	if in.NodeID != h.broker.InstanceID || in.Fingerprint != state.ApprovalFingerprint(a, h.broker.InstanceID) {
		return wire.ErrorResponse(r.ID, "scope_mismatch", "request snapshot or destination changed")
	}
	if r.Tool == "admin.approval.approve" && (a.Kind != "root" || a.Access != "read") && !in.StepUp {
		return wire.ErrorResponse(r.ID, "step_up_required", "fresh operator verification required")
	}
	r.AdminToken = h.broker.AdminToken
	r.Subject = operatorIdentity()
	r.Resource = a.Resource
	return h.broker.Handle(ctx, r)
}
func operatorToken() string { return os.Getenv("VPS_AGENT_OPERATOR_APPROVAL_TOKEN") }

func operatorIdentity() string {
	if id := os.Getenv("VPS_AGENT_OPERATOR_ID"); id != "" {
		if !utf8.ValidString(id) || unsafeDisplay(id) || len(id) > 1024 {
			return ""
		}
		return id
	}
	return "local-owner"
}
func unsafeDisplay(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			return true
		}
	}
	return false
}

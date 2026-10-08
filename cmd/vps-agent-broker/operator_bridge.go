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

 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

// operatorBridge exposes the minimal approval protocol, not the privileged
// Broker API. Crucially, a web-session compromise cannot approve shell.admin
// or any other capability elevation by submitting a hidden request ID.
type operatorBridge struct {
 broker *broker.Broker
 token string
}

func eligibleWebApproval(a state.Approval, expectedSubject string) bool {
 if a.Status != "pending" || !time.Now().Before(a.ExpiresAt) || a.Subject == "" ||
  (expectedSubject != "" && a.Subject != expectedSubject) {
  return false
 }
 if a.Kind == "root" {
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
  !strings.ContainsRune(string(decoded), 0)
}

func (h *operatorBridge) Handle(ctx context.Context, r wire.Request) wire.Response {
 switch r.Tool {
 case "admin.approval.list", "admin.approval.approve", "admin.approval.deny":
 default:
  return wire.ErrorResponse(r.ID, "permission_denied", "operator socket restricts operations")
 }
 if h.broker == nil || h.broker.State == nil || h.token == "" ||
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
    allowed = append(allowed, a)
   }
  }
  raw, err := json.Marshal(allowed)
  if err != nil {
   return wire.ErrorResponse(r.ID, "encode_error", "pending approvals unavailable")
  }
  return wire.Response{ID: r.ID, OK: true, Result: raw}
 }
 var in struct { RequestID string `json:"request_id"` }
 if err := json.Unmarshal(r.Args, &in); err != nil || !strings.HasPrefix(in.RequestID, "apr_") {
  return wire.ErrorResponse(r.ID, "invalid_args", "request ID invalid")
 }
 a, err := h.broker.State.GetApproval(ctx, in.RequestID)
 if err != nil || !eligibleWebApproval(a, h.broker.ExpectedSubject) {
  return wire.ErrorResponse(r.ID, "permission_denied", "request not eligible for web approval")
 }
 r.AdminToken = h.broker.AdminToken
 r.Subject = "operator-web"
 r.Resource = a.Resource
 return h.broker.Handle(ctx, r)
}
func operatorToken() string { return os.Getenv("VPS_AGENT_OPERATOR_APPROVAL_TOKEN") }

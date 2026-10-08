package main

import (
 "context"
 "crypto/subtle"
 "os"
 "strings"

 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

// operatorBridge exposes only the three approval operations on a distinct
// Unix socket. The UI service does not get the main Broker socket or Docker.
type operatorBridge struct {
 broker *broker.Broker
 token string
}
func (h *operatorBridge) Handle(ctx context.Context, r wire.Request) wire.Response {
 switch r.Tool {
 case "admin.approval.list", "admin.approval.approve", "admin.approval.deny":
 default:
  return wire.ErrorResponse(r.ID,"permission_denied","operator socket restricts operations")
 }
 if h.token == "" || len(r.AdminToken) != len(h.token) || subtle.ConstantTimeCompare([]byte(r.AdminToken), []byte(h.token)) != 1 {
  return wire.ErrorResponse(r.ID,"permission_denied","operator token invalid")
 }
 r.AdminToken = h.broker.AdminToken
 r.Subject = "operator-web"
 r.Resource = strings.TrimSpace(r.Resource)
 return h.broker.Handle(ctx,r)
}
func operatorToken() string { return os.Getenv("VPS_AGENT_OPERATOR_APPROVAL_TOKEN") }

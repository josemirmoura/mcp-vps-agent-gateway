package broker

import (
	"context"
	"encoding/json"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func (b *Broker) ownApproval(ctx context.Context, req wire.Request) (state.Approval, bool) {
	var in struct {
		RequestID string `json:"request_id"`
	}
	if b.State == nil || json.Unmarshal(req.Args, &in) != nil || in.RequestID == "" {
		return state.Approval{}, false
	}
	a, err := b.State.GetApproval(ctx, in.RequestID)
	return a, err == nil && a.Subject == req.Subject
}

func (b *Broker) approvalStatus(ctx context.Context, req wire.Request) wire.Response {
	a, okOwn := b.ownApproval(ctx, req)
	if !okOwn {
		return deny(req.ID, "not_found", "approval unavailable for this subject")
	}
	status, err := b.State.ApprovalAuthorityStatus(ctx, a)
	if err != nil {
		return deny(req.ID, "state_error", "approval outcome requires operator inspection")
	}
	resource, access := a.Resource, a.Access
	if a.Kind == "capability" && len(a.Capabilities) == 1 {
		if profile, path, valid := decodeSensitiveCapability(a.Capabilities[0]); valid {
			resource, access = path, profile
		}
	}
	return ok(req.ID, map[string]any{"request_id": a.ID, "status": status, "decision": a.Status,
		"node_id": b.InstanceID, "subject": a.Subject, "kind": a.Kind, "resource": resource,
		"access": access, "delegation_ttl_seconds": int64(a.TTL.Seconds()), "approval_expires_at": a.ExpiresAt, "retry_after_seconds": 3,
		"authoritative": true})
}

func (b *Broker) cancelApproval(ctx context.Context, req wire.Request) wire.Response {
	a, okOwn := b.ownApproval(ctx, req)
	if !okOwn {
		return deny(req.ID, "not_found", "approval unavailable for this subject")
	}
	_, err := b.State.ResolveApproval(ctx, a, "cancelled", state.AuditEvent{InstanceID: b.InstanceID, InstanceName: b.InstanceName, Subject: req.Subject, Tool: req.Tool, Resource: a.Resource})
	if err != nil {
		return deny(req.ID, "state_error", err.Error())
	}
	return ok(req.ID, map[string]any{"request_id": a.ID, "status": "cancelled"})
}

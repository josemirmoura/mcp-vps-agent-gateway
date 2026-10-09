package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func securityReviewFullBroker(t *testing.T) (*Broker, *state.Store) {
	t.Helper()
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	cfg := &policy.Config{
		Version: 1, Mode: "full", Enabled: true,
		Features: policy.Features{FullModeEnabled: true},
		Capabilities: []string{"shell.admin", "systemd.admin"},
		Grant: policy.GrantPolicy{Required: true, MaxTTLMinutes: 60, OutOfBandApproval: true, StepUpAuth: true},
		Network: policy.NetworkPolicy{Mode: "blocked", UnrestrictedRequiresSeparateApproval: true},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	return &Broker{Policy: cfg, State: store, AdminToken: "review-operator-secret", ExpectedSubject: "alice"}, store
}

// Seconds are untrusted integer input. Multiplication must never turn a large
// positive requested lifetime into an allowed short or permanent delegation.
func TestApprovalRequestLifetimeCannotOverflow(t *testing.T) {
	ctx := context.Background()
	for _, seconds := range []int64{1 << 55, (1 << 55) + 60, math.MaxInt64} {
		for _, kind := range []string{"root", "sensitive", "elevation"} {
			t.Run(fmt.Sprintf("%s/%d", kind, seconds), func(t *testing.T) {
				var b *Broker
				var store *state.Store
				tool, resource := "permissions.request_elevation", ""
				args := map[string]any{"ttl_seconds": seconds, "capabilities": []string{"shell.admin"}}
				if kind == "elevation" {
					b, store = securityReviewFullBroker(t)
				} else {
					var root string
					b, root, store = sensitiveTestBroker(t)
					t.Cleanup(func() { _ = store.Close() })
					resource = filepath.Join(root, "project")
					tool = "permissions.request_root_access"
					args = map[string]any{"root": resource, "access": "read", "ttl_seconds": seconds}
					if kind == "sensitive" {
						resource = filepath.Join(root, ".env")
						tool = "permissions.request_sensitive_access"
						args = map[string]any{"path": resource, "access": "read", "ttl_seconds": seconds}
						if err := os.WriteFile(resource, []byte("synthetic test value"), 0600); err != nil {
							t.Fatal(err)
						}
					} else if err := os.Mkdir(resource, 0750); err != nil {
						t.Fatal(err)
					}
				}
				payload, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				resp := b.Handle(ctx, wire.Request{ID: "oversized-lifetime", Subject: "alice", Tool: tool,
					Resource: resource, InvocationID: "overflow-request", Args: payload})
				if resp.OK || resp.Error == nil || resp.Error.Code != "invalid_ttl" {
					t.Fatalf("oversized lifetime was not rejected before persistence: %+v", resp)
				}
				pending, err := store.ListPendingApprovals(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if len(pending) != 0 {
					t.Fatalf("invalid lifetime created pending authority requests: %+v", pending)
				}
			})
		}
	}
}

func TestOperatorDecisionCannotGrantAfterSubjectBindingChanges(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0750); err != nil {
		t.Fatal(err)
	}
	a, err := store.CreateRootApproval(ctx, "alice", project, "work", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	b.ExpectedSubject = "bob"
	args, _ := json.Marshal(map[string]any{"request_id": a.ID})
	response := b.Handle(ctx, wire.Request{ID: "old-subject-decision", Subject: "owner", Tool: "admin.approval.approve", AdminToken: b.AdminToken, Args: args})
	if response.OK || response.Error == nil || response.Error.Code != "identity_mismatch" {
		t.Fatalf("decision crossed a changed machine/subject binding: %+v", response)
	}
	active, err := store.ListActiveRootDelegations(ctx, "alice")
	if err != nil || len(active) != 0 {
		t.Fatalf("prior authenticated subject received new authority: %+v, err=%v", active, err)
	}
	response = b.Handle(ctx, wire.Request{ID: "dispose-old-request", Subject: "owner", Tool: "admin.approval.deny", AdminToken: b.AdminToken, Args: args})
	if !response.OK {
		t.Fatalf("owner could not safely deny old subject's request: %+v", response)
	}
}

func TestBrokerDecisionRetainsOperatorSnapshotAndDestinationBinding(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()
	b.InstanceID = "node-a"
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0750); err != nil {
		t.Fatal(err)
	}
	a, err := store.CreateRootApproval(ctx, "alice", project, "read", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct { name, node, fingerprint string }{
		{"destination_changed", "node-b", state.ApprovalFingerprint(a, "node-a")},
		{"snapshot_from_other_node", "node-a", state.ApprovalFingerprint(a, "node-b")},
		{"displayed_resource_changed", "node-a", state.ApprovalFingerprint(state.Approval{ID: a.ID, Subject: a.Subject, Resource: "/opt/another-project"}, "node-a")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]any{"request_id": a.ID, "node_id": tc.node, "snapshot_hash": tc.fingerprint})
			response := b.Handle(ctx, wire.Request{ID: tc.name, Subject: "owner", Tool: "admin.approval.approve", AdminToken: b.AdminToken, Args: args})
			if response.OK || response.Error == nil || response.Error.Code != "scope_mismatch" {
				t.Fatalf("operator snapshot was dropped before Broker resolution: %+v", response)
			}
		})
	}
	args, _ := json.Marshal(map[string]any{"request_id": a.ID, "node_id": b.InstanceID, "snapshot_hash": state.ApprovalFingerprint(a, b.InstanceID)})
	response := b.Handle(ctx, wire.Request{ID: "matching-snapshot", Subject: "owner", Tool: "admin.approval.approve", AdminToken: b.AdminToken, Args: args})
	if !response.OK {
		t.Fatalf("matching independently authenticated operator snapshot failed: %+v", response)
	}
}

func TestApprovalRevalidatesCurrentFullCapabilityPolicy(t *testing.T) {
	ctx := context.Background()
	changes := []struct {
		name string
		apply func(*policy.Config)
	}{
		{"capability_removed", func(c *policy.Config) { c.Capabilities = []string{"systemd.admin"} }},
		{"full_disabled", func(c *policy.Config) { c.Enabled = false }},
		{"full_feature_disabled", func(c *policy.Config) { c.Features.FullModeEnabled = false }},
		{"mode_scoped", func(c *policy.Config) { c.Mode = "scoped" }},
	}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			b, store := securityReviewFullBroker(t)
			args, _ := json.Marshal(map[string]any{"capabilities": []string{"shell.admin"}, "ttl_seconds": 60})
			request := b.Handle(ctx, wire.Request{ID: "create", Subject: "alice", Tool: "permissions.request_elevation", Args: args})
			if !request.OK {
				t.Fatalf("baseline request failed: %+v", request)
			}
			var pending struct { RequestID string `json:"request_id"` }
			if err := json.Unmarshal(request.Result, &pending); err != nil || pending.RequestID == "" {
				t.Fatalf("invalid pending response: %s, err=%v", request.Result, err)
			}
			change.apply(b.Policy)
			decisionArgs, _ := json.Marshal(map[string]any{"request_id": pending.RequestID})
			response := b.Handle(ctx, wire.Request{ID: "approve-stale-policy", Subject: "owner", Tool: "admin.approval.approve", AdminToken: b.AdminToken, Args: decisionArgs})
			if response.OK {
				t.Fatal("operator approval resurrected a capability absent from current policy")
			}
			grants, err := store.ListActiveGrants(ctx, "alice")
			if err != nil || len(grants) != 0 {
				t.Fatalf("stale policy created authority: %+v, err=%v", grants, err)
			}
			a, err := store.GetApproval(ctx, pending.RequestID)
			if err != nil || a.Status != "pending" {
				t.Fatalf("failed approval mutated pending decision: %+v, err=%v", a, err)
			}
			// An operator can still dispose of a stale request without granting it.
			response = b.Handle(ctx, wire.Request{ID: "deny-stale-policy", Subject: "owner", Tool: "admin.approval.deny", AdminToken: b.AdminToken, Args: decisionArgs})
			if !response.OK {
				t.Fatalf("stale policy prevented a safe denial: %+v", response)
			}
		})
	}
}

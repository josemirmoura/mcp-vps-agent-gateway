package broker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func TestValidRequestTokenCannotAuthenticateOperator(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()
	if err := os.Mkdir(filepath.Join(root, "project"), 0750); err != nil {
		t.Fatal(err)
	}
	a, err := store.CreateRootApproval(ctx, "alice", filepath.Join(root, "project"), "read", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, err := b.rootApprovalToken(a)
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{"request_id": a.ID, "approval_token": token, "decision": "approve"})
	response := b.Handle(ctx, wire.Request{ID: "model-root", Subject: "alice", Tool: "permissions.confirm_root_access", Args: args})
	if response.OK || response.Error.Code != "operator_required" {
		t.Fatalf("request token became owner credential: %+v", response)
	}
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	requestArgs, _ := json.Marshal(map[string]any{"path": path, "access": "read", "ttl_seconds": 60})
	pending := b.Handle(ctx, wire.Request{ID: "request-sensitive", Subject: "alice", Tool: "permissions.request_sensitive_access", Resource: path, InvocationID: "sensitive-adaptive", Args: requestArgs})
	if !pending.OK {
		t.Fatalf("request failed: %+v", pending)
	}
	var value map[string]any
	if err := json.Unmarshal(pending.Result, &value); err != nil {
		t.Fatal(err)
	}
	value["decision"] = "approve"
	args, _ = json.Marshal(value)
	response = b.Handle(ctx, wire.Request{ID: "model-sensitive", Subject: "alice", Tool: "permissions.confirm_sensitive_access", Args: args})
	if response.OK || response.Error.Code != "operator_required" {
		t.Fatal("valid sensitive request token self-approved")
	}
	roots, _ := store.ListActiveRootDelegations(ctx, "alice")
	grants, _ := store.ListActiveGrants(ctx, "alice")
	if len(roots) != 0 || len(grants) != 0 {
		t.Fatal("self-approval created authority")
	}
}

func TestApprovalStatusAndCancellationSubjectIsolation(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()
	b.InstanceID = "node-a"
	a, err := store.CreateRootApproval(ctx, "alice", root, "read", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{"request_id": a.ID})
	for _, tool := range []string{"permissions.approval_status", "permissions.cancel_approval"} {
		if response := b.Handle(ctx, wire.Request{ID: "foreign", Subject: "bob", Tool: tool, Args: args}); response.OK {
			t.Fatal("foreign subject saw/changed request")
		}
	}
	// Even a multi-subject Broker has per-request ownership checks.
	b.ExpectedSubject = ""
	if response := b.Handle(ctx, wire.Request{ID: "idor", Subject: "bob", Tool: "permissions.cancel_approval", Args: args}); response.OK {
		t.Fatal("unbound Broker allowed IDOR")
	}
	response := b.Handle(ctx, wire.Request{ID: "status", Subject: "alice", Tool: "permissions.approval_status", Args: args})
	var data map[string]any
	if !response.OK || json.Unmarshal(response.Result, &data) != nil || data["node_id"] != "node-a" || data["status"] != "pending" {
		t.Fatalf("invalid status: %+v", response)
	}
	response = b.Handle(ctx, wire.Request{ID: "cancel", Subject: "alice", Tool: "permissions.cancel_approval", Args: args})
	if !response.OK {
		t.Fatalf("cancel failed: %+v", response)
	}
	if response = b.Handle(ctx, wire.Request{ID: "late-owner", Subject: "owner", Tool: "admin.approval.approve", AdminToken: b.AdminToken, Args: args}); response.OK {
		t.Fatal("cancelled request was approved")
	}
}

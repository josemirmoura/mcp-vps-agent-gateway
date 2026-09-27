package broker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/jobs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sandbox"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func TestElevationRequiresOutOfBandOperatorDecision(t *testing.T) {
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cfg := &policy.Config{
		Version: 1,
		Mode:    "full",
		Enabled: true,
		Features: policy.Features{
			FullModeEnabled: true,
		},
		Capabilities: []string{"shell.admin", "systemd.admin"},
		Grant: policy.GrantPolicy{
			Required:          true,
			MaxTTLMinutes:     60,
			OutOfBandApproval: true,
			StepUpAuth:        true,
		},
		Network: policy.NetworkPolicy{
			Mode: "blocked",
			UnrestrictedRequiresSeparateApproval: true,
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	b := &Broker{Policy: cfg, State: store, AdminToken: "operator-secret"}
	args, _ := json.Marshal(map[string]any{
		"capabilities": []string{"shell.admin"},
		"ttl_seconds":  60,
	})
	request := b.Handle(context.Background(), wire.Request{
		ID: "1", Subject: "alice", Tool: "permissions.request_elevation", Args: args,
	})
	if !request.OK {
		t.Fatalf("elevation request failed: %+v", request)
	}
	var pending struct {
		RequestID string `json:"request_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(request.Result, &pending); err != nil {
		t.Fatal(err)
	}
	if pending.RequestID == "" || pending.Status != "pending" {
		t.Fatalf("unexpected pending response: %+v", pending)
	}

	approveArgs, _ := json.Marshal(map[string]any{"request_id": pending.RequestID})

	// The MCP caller does not possess an operator secret.
	wrong := b.Handle(context.Background(), wire.Request{
		ID: "2", Subject: "alice", Tool: "admin.approval.approve",
		AdminToken: "wrong", Args: approveArgs,
	})
	if wrong.OK || wrong.Error == nil || wrong.Error.Code != "permission_denied" {
		t.Fatalf("self/unauthorized approval succeeded: %+v", wrong)
	}

	approved := b.Handle(context.Background(), wire.Request{
		ID: "3", Tool: "admin.approval.approve",
		AdminToken: "operator-secret", Args: approveArgs,
	})
	if !approved.OK {
		t.Fatalf("operator approval failed: %+v", approved)
	}
	var result struct {
		GrantID string `json:"grant_id"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal(approved.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.GrantID == "" || result.Subject != "alice" {
		t.Fatalf("unexpected grant: %+v", result)
	}
	ok, err := store.ValidateGrant(context.Background(), result.GrantID, "alice", "shell.admin")
	if err != nil || !ok {
		t.Fatalf("issued grant invalid: ok=%v err=%v", ok, err)
	}

	// A capability absent from the policy is denied before an approval is created.
	netArgs, _ := json.Marshal(map[string]any{
		"capabilities": []string{"network.unrestricted"},
		"ttl_seconds":  60,
	})
	netReq := b.Handle(context.Background(), wire.Request{
		ID: "4", Subject: "alice", Tool: "permissions.request_elevation", Args: netArgs,
	})
	if netReq.OK {
		t.Fatal("network.unrestricted was implicitly grantable")
	}

	// Expiry behavior remains server-side.
	short, err := store.IssueGrant(context.Background(), "alice", []string{"shell.admin"}, 2*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Millisecond)
	ok, err = store.ValidateGrant(context.Background(), short.ID, "alice", "shell.admin")
	if err != nil || ok {
		t.Fatalf("expired grant accepted: ok=%v err=%v", ok, err)
	}
}

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


type revokeTestRunner struct {
	cancelled []string
}

func (r *revokeTestRunner) Start(_ context.Context, spec sandbox.Spec) (jobs.Job, error) {
	return jobs.Job{ID: spec.Unit, Unit: spec.Unit, State: "running", Deadline: time.Now().Add(spec.Runtime)}, nil
}

func (r *revokeTestRunner) Status(context.Context, jobs.Job) (string, error) {
	return "active", nil
}

func (r *revokeTestRunner) Tail(context.Context, jobs.Job, int) (string, error) {
	return "", nil
}

func (r *revokeTestRunner) Cancel(_ context.Context, job jobs.Job) error {
	r.cancelled = append(r.cancelled, job.ID)
	return nil
}

func TestRevokeAllCancelsAlreadyRunningElevatedJobs(t *testing.T) {
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	runner := &revokeTestRunner{}
	manager := &jobs.Manager{State: store, Runner: runner}

	grant, err := store.IssueGrant(ctx, "alice", []string{"shell.admin"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := manager.Start(ctx, "alice", "shell.exec_admin", "/srv/app", grant.ID, sandbox.Spec{
		Unit: "job-revoke-active", Command: "true", CWD: "/srv/app",
		Runtime: time.Minute, NetworkMode: "blocked",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Status(ctx, "alice", rec.ID); err != nil {
		t.Fatal(err)
	}
	rec, err = store.GetJob(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != "active" {
		t.Fatalf("job state=%q, want active", rec.State)
	}

	b := &Broker{State: store, Jobs: manager, AdminToken: "operator-secret"}
	resp := b.Handle(ctx, wire.Request{
		ID: "revoke", Tool: "admin.revoke_all", AdminToken: "operator-secret",
	})
	if !resp.OK {
		t.Fatalf("revoke-all failed: %+v", resp)
	}
	var result struct {
		Revoked               bool `json:"revoked"`
		CancelledElevatedJobs int  `json:"cancelled_elevated_jobs"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatal(err)
	}
	if !result.Revoked || result.CancelledElevatedJobs != 1 {
		t.Fatalf("unexpected revoke result: %+v", result)
	}
	if len(runner.cancelled) != 1 || runner.cancelled[0] != rec.ID {
		t.Fatalf("runner cancellations=%v, want [%s]", runner.cancelled, rec.ID)
	}
	valid, err := store.ValidateGrant(ctx, grant.ID, "alice", "shell.admin")
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("grant remained valid after revoke-all")
	}
	rec, err = store.GetJob(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != "cancelled" {
		t.Fatalf("job state=%q, want cancelled", rec.State)
	}
}

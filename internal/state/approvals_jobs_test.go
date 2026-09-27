package state

import (
	"context"
	"testing"
	"time"
)

func TestApprovalLifecycle(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	a, err := s.CreateApproval(ctx, "alice", []string{"shell.admin"}, time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := s.ListPendingApprovals(ctx)
	if err != nil || len(pending) != 1 || pending[0].ID != a.ID {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
	approved, err := s.DecideApproval(ctx, a.ID, "approved")
	if err != nil || approved.Status != "approved" {
		t.Fatalf("approved=%+v err=%v", approved, err)
	}
	if _, err := s.DecideApproval(ctx, a.ID, "approved"); err == nil {
		t.Fatal("second decision should fail")
	}
}

func TestJobPersistence(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	deadline := time.Now().Add(time.Minute)
	j := JobRecord{
		ID: "job-1", Subject: "alice", Tool: "shell.exec", Resource: "/srv/app",
		UnitName: "vps-agent-job-1", State: "running", Deadline: deadline,
	}
	if err := s.CreateJob(ctx, j); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetJob(ctx, j.ID)
	if err != nil || got.State != "running" || got.Subject != "alice" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	code := 0
	if err := s.UpdateJobState(ctx, j.ID, "done", "ok", &code); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetJob(ctx, j.ID)
	if got.State != "done" || got.ExitCode == nil || *got.ExitCode != 0 {
		t.Fatalf("updated=%+v", got)
	}
}


func TestRevokeAllInvalidatesGrantsAndPendingApprovals(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	grant, err := s.IssueGrant(ctx, "alice", []string{"shell.admin"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := s.CreateApproval(ctx, "alice", []string{"shell.admin"}, time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.RevokeAll(ctx); err != nil {
		t.Fatal(err)
	}
	valid, err := s.ValidateGrant(ctx, grant.ID, "alice", "shell.admin")
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("revoke-all left an existing grant valid")
	}
	pending, err := s.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("revoke-all left pending approvals: %+v", pending)
	}
	got, err := s.GetApproval(ctx, approval.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "revoked" {
		t.Fatalf("approval status=%q, want revoked", got.Status)
	}
	if _, err := s.DecideApproval(ctx, approval.ID, "approved"); err == nil {
		t.Fatal("revoked approval could still be approved")
	}
}

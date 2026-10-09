package state

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestApprovalScopeFingerprintRejectsChangedSecurityFields(t *testing.T) {
	ctx := context.Background()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, err := s.CreateRootApproval(ctx, "alice", "/opt/project", "read", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	changes := []struct {
		name string
		apply func(*Approval)
	}{
		{"subject", func(a *Approval) { a.Subject = "mallory" }},
		{"resource", func(a *Approval) { a.Resource = "/opt/another-project" }},
		{"access", func(a *Approval) { a.Access = "work" }},
		{"capabilities", func(a *Approval) { a.Capabilities = []string{"shell.admin"} }},
		{"ttl", func(a *Approval) { a.TTL = 2 * time.Minute }},
		{"permanent", func(a *Approval) { a.TTL = 0 }},
		{"creation_time", func(a *Approval) { a.CreatedAt = a.CreatedAt.Add(time.Second) }},
		{"expiry", func(a *Approval) { a.ExpiresAt = a.ExpiresAt.Add(time.Second) }},
		{"kind", func(a *Approval) { a.Kind = "capability" }},
	}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			snapshot := a
			change.apply(&snapshot)
			if _, err := s.ResolveApproval(ctx, snapshot, "approved", AuditEvent{InstanceID: "node-a", Subject: "owner"}); err == nil {
				t.Fatal("changed displayed scope created authority")
			}
			stored, err := s.GetApproval(ctx, a.ID)
			if err != nil || stored.Status != "pending" {
				t.Fatalf("scope mismatch changed pending request: %+v, err=%v", stored, err)
			}
		})
	}
	roots, err := s.ListActiveRootDelegations(ctx, "alice")
	if err != nil || len(roots) != 0 {
		t.Fatalf("scope mismatch issued root authority: %+v, err=%v", roots, err)
	}
	events, err := s.ListAuditAfter(ctx, 0, 100)
	if err != nil || len(events) != 0 {
		t.Fatalf("rejected scope escaped atomic resolution: %+v, err=%v", events, err)
	}
	if ApprovalFingerprint(a, "node-a") == ApprovalFingerprint(a, "node-b") {
		t.Fatal("request fingerprint can be replayed on a different destination")
	}
}

func TestApprovalAuditFailurePreservesExistingRootAuthority(t *testing.T) {
	ctx := context.Background()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	old, err := s.IssueRootDelegation(ctx, "alice", "/opt/project", "read", "historical-approval", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.CreateRootApproval(ctx, "alice", "/opt/project", "work", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER review_reject_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'synthetic audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveApproval(ctx, a, "approved", AuditEvent{InstanceID: "node-a", Subject: "owner", Tool: "admin.approval.approve"}); err == nil {
		t.Fatal("audit failure was ignored")
	}
	active, err := s.ListActiveRootDelegations(ctx, "alice")
	if err != nil || len(active) != 1 || active[0].ID != old.ID || active[0].Access != "read" {
		t.Fatalf("failed resolution revoked/replaced prior authority: %+v, err=%v", active, err)
	}
	stored, err := s.GetApproval(ctx, a.ID)
	if err != nil || stored.Status != "pending" {
		t.Fatalf("audit failure committed decision: %+v, err=%v", stored, err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER review_reject_audit`); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.ResolveApproval(ctx, a, "approved", AuditEvent{InstanceID: "node-a", Subject: "owner", Tool: "admin.approval.approve"})
	if err != nil || resolved.Root == nil {
		t.Fatalf("valid retry after rollback failed: %+v, err=%v", resolved, err)
	}
	active, err = s.ListActiveRootDelegations(ctx, "alice")
	if err != nil || len(active) != 1 || active[0].ID != resolved.Root.ID || active[0].Access != "work" {
		t.Fatalf("committed replacement did not have exactly one active root: %+v, err=%v", active, err)
	}
	if err := s.VerifyAudit(ctx); err != nil {
		t.Fatal(err)
	}
	events, err := s.ListAuditAfter(ctx, 0, 10)
	if err != nil || len(events) != 1 || events[0].Event.GrantID != resolved.Root.ID || events[0].Event.ApprovalID != a.ID || events[0].Event.Requester != "alice" || events[0].Event.Subject != "owner" {
		t.Fatalf("authority lacks committed operator/requester provenance: %+v, err=%v", events, err)
	}
}

func TestApprovalDecisionAtomicAcrossIndependentStoreConnections(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "state.db")
	s1, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()
	s2, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	a, err := s1.CreateRootApproval(ctx, "alice", "/opt/project", "work", time.Minute, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var committed atomic.Int32
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 10; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			s := s1
			if i%2 != 0 {
				s = s2
			}
			if _, err := s.ResolveApproval(ctx, a, "approved", AuditEvent{InstanceID: "node-a", Subject: "owner"}); err == nil {
				committed.Add(1)
			}
		}(i)
	}
	close(start)
	workers.Wait()
	if committed.Load() != 1 {
		t.Fatalf("independent database connections committed %d decisions", committed.Load())
	}
	active, err := s1.ListActiveRootDelegations(ctx, "alice")
	if err != nil || len(active) != 1 || active[0].ApprovalID != a.ID {
		t.Fatalf("concurrent replay created duplicate/lost authority: %+v, err=%v", active, err)
	}
	events, err := s1.ListAuditAfter(ctx, 0, 100)
	if err != nil || len(events) != 1 || events[0].Event.GrantID != active[0].ID {
		t.Fatalf("concurrent decisions broke atomic audit: %+v, err=%v", events, err)
	}
	if err := s1.VerifyAudit(ctx); err != nil {
		t.Fatal(err)
	}
}

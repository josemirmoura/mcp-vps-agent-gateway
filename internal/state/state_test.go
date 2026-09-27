package state

import (
	"context"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestIdempotencyJournal(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	d, _, err := s.BeginOperation(ctx, "inv1", "alice", "file.write", "hash1")
	if err != nil || d != OperationExecute {
		t.Fatalf("first decision=%s err=%v", d, err)
	}
	d, _, err = s.BeginOperation(ctx, "inv1", "alice", "file.write", "hash1")
	if err != nil || d != OperationReconcile {
		t.Fatalf("pending retry=%s err=%v", d, err)
	}
	if err := s.CompleteOperation(ctx, "inv1", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	d, cached, err := s.BeginOperation(ctx, "inv1", "alice", "file.write", "hash1")
	if err != nil || d != OperationCached || len(cached) == 0 {
		t.Fatalf("cached=%s err=%v response=%s", d, err, cached)
	}
	d, _, err = s.BeginOperation(ctx, "inv1", "alice", "file.write", "different")
	if err != nil || d != OperationConflict {
		t.Fatalf("conflict=%s err=%v", d, err)
	}
}

func TestFencingTokenBlocksStaleRelease(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	a, err := s.AcquireLock(ctx, "stack:x", "A", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Millisecond)
	b, err := s.AcquireLock(ctx, "stack:x", "B", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if b.Token <= a.Token {
		t.Fatalf("token did not advance: A=%d B=%d", a.Token, b.Token)
	}
	released, err := s.ReleaseLock(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if released {
		t.Fatal("stale owner released a newer lock")
	}
}

func TestGrantSubjectCapabilityAndExpiry(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	g, err := s.IssueGrant(ctx, "alice", []string{"shell.admin"}, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := s.ValidateGrant(ctx, g.ID, "alice", "shell.admin")
	if err != nil || !ok {
		t.Fatalf("valid grant rejected: ok=%v err=%v", ok, err)
	}
	ok, _ = s.ValidateGrant(ctx, g.ID, "bob", "shell.admin")
	if ok {
		t.Fatal("foreign subject reused grant")
	}
	ok, _ = s.ValidateGrant(ctx, g.ID, "alice", "network.unrestricted")
	if ok {
		t.Fatal("unapproved network capability leaked")
	}
	time.Sleep(8 * time.Millisecond)
	ok, _ = s.ValidateGrant(ctx, g.ID, "alice", "shell.admin")
	if ok {
		t.Fatal("expired grant accepted")
	}
}

func TestAuditTamperDetected(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	for i := 0; i < 10; i++ {
		if _, err := s.AppendAudit(ctx, AuditEvent{Subject: "alice", Tool: "test", Decision: "allow"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.VerifyAudit(ctx); err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE audit_events SET event_json='{"tampered":true}' WHERE seq=5`); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyAudit(ctx); err == nil {
		t.Fatal("tampered chain was not detected")
	}
}

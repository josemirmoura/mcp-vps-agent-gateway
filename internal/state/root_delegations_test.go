package state

import (
	"context"
	"testing"
	"time"
)

func TestRootDelegationLifecycle(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	d, err := s.IssueRootDelegation(ctx, "alice", "/opt/project-a", "work", "apr_1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if d.ExpiresAt != nil {
		t.Fatal("permanent delegation unexpectedly expires")
	}
	active, err := s.ListActiveRootDelegations(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Root != "/opt/project-a" || active[0].Access != "work" {
		t.Fatalf("unexpected active delegations: %+v", active)
	}

	replacement, err := s.IssueRootDelegation(ctx, "alice", "/opt/project-a", "compose", "apr_2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.ExpiresAt == nil {
		t.Fatal("temporary replacement delegation has no expiry")
	}
	active, err = s.ListActiveRootDelegations(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Access != "compose" {
		t.Fatalf("replacement did not supersede prior delegation: %+v", active)
	}

	n, err := s.RevokeRootDelegation(ctx, "alice", "/opt/project-a")
	if err != nil || n != 1 {
		t.Fatalf("revoke count=%d err=%v", n, err)
	}
	active, err = s.ListActiveRootDelegations(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("revoked delegation remained active: %+v", active)
	}
}

func TestRootDelegationExpiryAndSubjectIsolation(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	if _, err := s.IssueRootDelegation(ctx, "alice", "/opt/a", "read", "apr_a", 2*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssueRootDelegation(ctx, "bob", "/opt/b", "work", "apr_b", 0); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Millisecond)

	alice, err := s.ListActiveRootDelegations(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(alice) != 0 {
		t.Fatalf("expired delegation remained active: %+v", alice)
	}
	bob, err := s.ListActiveRootDelegations(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if len(bob) != 1 || bob[0].Root != "/opt/b" {
		t.Fatalf("subject isolation failed: %+v", bob)
	}
}

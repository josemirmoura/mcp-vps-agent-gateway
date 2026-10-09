package state

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestApprovalCapacityAtomicBoundAndExpiryReleasesCapacity(t *testing.T) {
	ctx := context.Background()
	s, err := Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer s.Close()
	var created atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < MaxPendingApprovalsPerSubject+16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.CreateRootApproval(ctx, "alice", fmt.Sprintf("/opt/project-%d", i), "read", time.Minute, time.Minute)
			if err == nil { created.Add(1) } else if !errors.Is(err, ErrApprovalQueueFull) { t.Error(err) }
		}(i)
	}
	wg.Wait()
	if int(created.Load()) != MaxPendingApprovalsPerSubject { t.Fatalf("active capacity=%d", created.Load()) }
	if _, err := s.db.Exec(`UPDATE approvals SET expires_at=?`, time.Now().Add(-time.Second).UnixNano()); err != nil { t.Fatal(err) }
	rows, err := s.ListPendingApprovals(ctx)
	if err != nil || len(rows) != 0 { t.Fatalf("expired requests remained in active queue: %d %v", len(rows), err) }
	if _, err := s.CreateRootApproval(ctx, "alice", "/opt/retry", "read", time.Minute, time.Minute); err != nil { t.Fatal(err) }
}

func TestApprovalCapacityGlobalBoundAndCancellationReleasesSlot(t *testing.T) {
	ctx := context.Background()
	s, err := Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer s.Close()
	var first Approval
	for i := 0; i < MaxPendingApprovals; i++ {
		a, err := s.CreateRootApproval(ctx, fmt.Sprintf("subject-%d", i), "/opt/project", "read", time.Minute, time.Minute)
		if err != nil { t.Fatal(err) }
		if i == 0 { first = a }
	}
	if _, err := s.CreateRootApproval(ctx, "one-more", "/opt/project", "read", time.Minute, time.Minute); !errors.Is(err, ErrApprovalQueueFull) { t.Fatalf("global bound bypassed: %v", err) }
	if _, err := s.ResolveApproval(ctx, first, "cancelled", AuditEvent{Subject:first.Subject}); err != nil { t.Fatal(err) }
	if _, err := s.CreateRootApproval(ctx, "one-more", "/opt/project", "read", time.Minute, time.Minute); err != nil { t.Fatal(err) }
}

package broker

import (
	"context"
	"errors"
	"testing"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func auditRestartRequest() wire.Request {
	return wire.Request{
		ID: "audited-restart", Subject: "alice", Tool: "service.restart",
		Resource: "vps-agent-test.service", InvocationID: "audited-restart-invocation",
	}
}

func TestAuditMutationIntentPrecedesServiceEffect(t *testing.T) {
	b, svc, store, _ := testBroker(t)
	req := auditRestartRequest()
	var seen []string
	b.auditAppendForTest = func(ctx context.Context, ev state.AuditEvent) (string, error) {
		seen = append(seen, ev.Decision)
		if ev.Decision == "intent" && svc.restarts != 0 {
			t.Fatal("service mutated before audit intent")
		}
		return store.AppendAudit(ctx, ev)
	}
	if resp := b.Handle(context.Background(), req); !resp.OK {
		t.Fatalf("expected successful audited restart: %+v", resp)
	}
	if svc.restarts != 1 {
		t.Fatalf("expected exactly one service restart, got %d", svc.restarts)
	}
	if len(seen) != 2 || seen[0] != "intent" || seen[1] != "allow" {
		t.Fatalf("unexpected audit sequence: %v", seen)
	}
	events, err := store.ListAuditAfter(context.Background(), 0, 10)
	if err != nil || len(events) != 2 {
		t.Fatalf("missing durable audit events: count=%d err=%v", len(events), err)
	}
	if events[0].Event.ActionID != req.InvocationID || events[0].Event.Decision != "intent" ||
		events[1].Event.Decision != "allow" {
		t.Fatalf("intent/outcome not linked to invocation: %+v", events)
	}
	if err := store.VerifyAudit(context.Background()); err != nil {
		t.Fatalf("audit chain broken: %v", err)
	}
}

func TestAuditIntentWriteFailureBlocksMutationBeforeService(t *testing.T) {
	b, svc, _, _ := testBroker(t)
	b.auditAppendForTest = func(context.Context, state.AuditEvent) (string, error) {
		return "", errors.New("synthetic sqlite audit write failure")
	}
	resp := b.Handle(context.Background(), auditRestartRequest())
	if resp.OK || resp.Error == nil || resp.Error.Code != "audit_unavailable" {
		t.Fatalf("missing pre-execution fail-closed result: %+v", resp)
	}
	if svc.restarts != 0 {
		t.Fatalf("service mutated despite failed durable intent: %d", svc.restarts)
	}
}

func TestAuditOutcomeFailureReportsUncertaintyAndPreservesIntent(t *testing.T) {
	b, svc, store, _ := testBroker(t)
	req := auditRestartRequest()
	calls := 0
	b.auditAppendForTest = func(ctx context.Context, ev state.AuditEvent) (string, error) {
		calls++
		if calls == 2 {
			return "", errors.New("synthetic final audit write failure")
		}
		return store.AppendAudit(ctx, ev)
	}
	resp := b.Handle(context.Background(), req)
	if resp.OK || resp.Error == nil || resp.Error.Code != "reconcile_required" {
		t.Fatalf("an unaudited mutation was reported as success: %+v", resp)
	}
	if svc.restarts != 1 {
		t.Fatalf("expected the already started external effect exactly once: %d", svc.restarts)
	}
	events, err := store.ListAuditAfter(context.Background(), 0, 10)
	if err != nil || len(events) != 1 || events[0].Event.Decision != "intent" {
		t.Fatalf("durable uncertain intent missing: events=%+v err=%v", events, err)
	}
	if err := store.VerifyAudit(context.Background()); err != nil {
		t.Fatalf("durable intent audit chain invalid: %v", err)
	}

	// The same invocation is handled by the existing idempotency journal;
	// a client's reconciliation/retry must not duplicate the service restart.
	b.auditAppendForTest = nil
	if repeat := b.Handle(context.Background(), req); !repeat.OK {
		t.Fatalf("existing cached operation could not be reconciled: %+v", repeat)
	}
	if svc.restarts != 1 {
		t.Fatalf("side effect duplicated after uncertain response: %d", svc.restarts)
	}
}

func TestMissingAuditStoreBlocksNewExternalMutation(t *testing.T) {
	b, svc, _, _ := testBroker(t)
	b.State = nil
	resp := b.Handle(context.Background(), auditRestartRequest())
	if resp.OK || resp.Error == nil || resp.Error.Code != "audit_unavailable" {
		t.Fatalf("mutation without state was not denied: %+v", resp)
	}
	if svc.restarts != 0 {
		t.Fatalf("mutation executed with unavailable audit store: %d", svc.restarts)
	}
}

func TestUnknownToolIsNotMisclassifiedAsReadOnly(t *testing.T) {
	if auditedReadOnlyTool("new.privileged.method") {
		t.Fatal("an unknown operation was silently exempted from intent journaling")
	}
	if !auditedReadOnlyTool("service.status") || auditedReadOnlyTool("service.restart") {
		t.Fatal("read-only and mutating service tools were misclassified")
	}
}

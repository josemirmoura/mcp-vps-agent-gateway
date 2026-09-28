package broker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type fakeServices struct {
	mu       sync.Mutex
	restarts int
}

func (f *fakeServices) List(context.Context) ([]ServiceInfo, error) {
	return []ServiceInfo{{Name: "vps-agent-test.service", Active: "active"}}, nil
}
func (f *fakeServices) Status(context.Context, string) (string, error) { return "active", nil }
func (f *fakeServices) Logs(context.Context, string, int) (string, error) { return "log", nil }
func (f *fakeServices) Start(context.Context, string) (string, error) { return "active", nil }
func (f *fakeServices) Stop(context.Context, string) (string, error) { return "inactive", nil }
func (f *fakeServices) Reload(context.Context, string) (string, error) { return "active", nil }
func (f *fakeServices) Enable(context.Context, string) (string, error) { return "enabled", nil }
func (f *fakeServices) Disable(context.Context, string) (string, error) { return "disabled", nil }
func (f *fakeServices) Restart(context.Context, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restarts++
	return "active", nil
}

func testBroker(t *testing.T) (*Broker, *fakeServices, *state.Store, string) {
	t.Helper()
	root := t.TempDir()
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	fs, err := securefs.New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	svc := &fakeServices{}
	cfg := &policy.Config{
		Version: 1, Mode: "scoped",
		Filesystem: policy.FilesystemPolicy{Read: []string{root}, Write: []string{root}},
		Services: policy.ResourcePolicy{
			Inspect: []string{"vps-agent-test.service"},
			Manage:  []string{"vps-agent-test.service"},
			Actions: []string{"status", "restart"},
		},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	return &Broker{Policy: cfg, FS: fs, State: store, Services: svc, AdminToken: "operator-secret"}, svc, store, root
}

func TestAdminHealthReportsActiveJobsAndRequiresOperatorToken(t *testing.T) {
	b, _, store, _ := testBroker(t)
	ctx := context.Background()

	denied := b.Handle(ctx, wire.Request{
		ID: "health-denied", Tool: "admin.health", AdminToken: "wrong",
	})
	if denied.OK || denied.Error == nil || denied.Error.Code != "permission_denied" {
		t.Fatalf("invalid operator token was accepted: %+v", denied)
	}

	if err := store.CreateJob(ctx, state.JobRecord{
		ID: "job-active", Subject: "alice", Tool: "shell.exec",
		State: "running", Deadline: time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	resp := b.Handle(ctx, wire.Request{
		ID: "health-ok", Tool: "admin.health", AdminToken: "operator-secret",
	})
	if !resp.OK {
		t.Fatalf("admin health failed: %+v", resp)
	}
	var out struct {
		OK             bool `json:"ok"`
		AuditChainOK   bool `json:"audit_chain_ok"`
		StateConfigured bool `json:"state_configured"`
		ActiveJobs     int  `json:"active_jobs"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || !out.AuditChainOK || !out.StateConfigured || out.ActiveJobs != 1 {
		t.Fatalf("unexpected health snapshot: %+v", out)
	}
}

func TestConfusedDeputyDenied(t *testing.T) {
	b, _, _, _ := testBroker(t)
	resp := b.Handle(context.Background(), wire.Request{
		ID: "1", Subject: "alice", Tool: "service.restart",
		Resource: "postgres.service", InvocationID: "inv-1",
	})
	if resp.OK || resp.Error == nil || resp.Error.Code != "permission_denied" {
		t.Fatalf("forbidden service allowed: %+v", resp)
	}
}

func TestRestartIdempotency(t *testing.T) {
	b, svc, _, _ := testBroker(t)
	req := wire.Request{
		ID: "1", Subject: "alice", Tool: "service.restart",
		Resource: "vps-agent-test.service", InvocationID: "stable-invocation",
	}
	if resp := b.Handle(context.Background(), req); !resp.OK {
		t.Fatalf("first restart failed: %+v", resp)
	}
	req.ID = "2"
	if resp := b.Handle(context.Background(), req); !resp.OK {
		t.Fatalf("cached restart failed: %+v", resp)
	}
	if svc.restarts != 1 {
		t.Fatalf("restart side effect repeated %d times", svc.restarts)
	}
}

func TestPendingRestartRequiresReconciliation(t *testing.T) {
	b, svc, store, _ := testBroker(t)
	req := wire.Request{
		ID: "1", Subject: "alice", Tool: "service.restart",
		Resource: "vps-agent-test.service", InvocationID: "uncertain",
	}
	h, _ := state.HashRequest(map[string]any{"subject": req.Subject, "tool": req.Tool, "service": req.Resource, "action": "restart"})
	if _, _, err := store.BeginOperation(context.Background(), req.InvocationID, req.Subject, req.Tool, h); err != nil {
		t.Fatal(err)
	}
	resp := b.Handle(context.Background(), req)
	if resp.OK || resp.Error == nil || resp.Error.Code != "reconcile_required" {
		t.Fatalf("expected reconcile_required: %+v", resp)
	}
	if svc.restarts != 0 {
		t.Fatal("uncertain operation was blindly retried")
	}
}

func TestFileWriteAndEscape(t *testing.T) {
	b, _, _, root := testBroker(t)
	args, _ := json.Marshal(map[string]any{"content": "hello"})
	target := filepath.Join(root, "ok.txt")
	resp := b.Handle(context.Background(), wire.Request{
		ID: "1", Subject: "alice", Tool: "file.write", Resource: target,
		InvocationID: "write-1", Args: args,
	})
	if !resp.OK {
		t.Fatalf("write failed: %+v", resp)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "hello" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	resp = b.Handle(context.Background(), wire.Request{
		ID: "2", Subject: "alice", Tool: "file.read",
		Resource: filepath.Join(root, "..", "secret"),
	})
	if resp.OK {
		t.Fatal("escaped read was allowed")
	}
}


func TestExpectedSubjectCannotBeForgedByGateway(t *testing.T) {
	b, _, _, _ := testBroker(t)
	b.ExpectedSubject = "alice"

	allowed := b.Handle(context.Background(), wire.Request{
		ID: "subject-ok", Subject: "alice", Tool: "service.status",
		Resource: "vps-agent-test.service",
	})
	if !allowed.OK {
		t.Fatalf("expected configured subject to pass: %+v", allowed)
	}

	denied := b.Handle(context.Background(), wire.Request{
		ID: "subject-forged", Subject: "root-admin", Tool: "service.status",
		Resource: "vps-agent-test.service",
	})
	if denied.OK || denied.Error == nil || denied.Error.Code != "identity_mismatch" {
		t.Fatalf("forged subject was not rejected: %+v", denied)
	}
}


func TestMkdirCreatesAuthorizedRootAndIsIdempotent(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "new-root")
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	fs, err := securefs.New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &policy.Config{
		Version: 1, Mode: "scoped",
		Filesystem: policy.FilesystemPolicy{Read: []string{root}, Write: []string{root}},
		Network: policy.NetworkPolicy{Mode: "blocked"},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	b := &Broker{Policy: cfg, FS: fs, State: store, ExpectedSubject: "alice"}

	req := wire.Request{
		ID: "mkdir-1", Subject: "alice", Tool: "file.mkdir",
		Resource: root, InvocationID: "mkdir-op",
	}
	if resp := b.Handle(context.Background(), req); !resp.OK {
		t.Fatalf("mkdir failed: %+v", resp)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("directory missing: info=%v err=%v", info, err)
	}

	req.ID = "mkdir-2"
	if resp := b.Handle(context.Background(), req); !resp.OK {
		t.Fatalf("cached mkdir failed: %+v", resp)
	}
}

func TestMkdirOutsideAuthorizedRootDenied(t *testing.T) {
	b, _, _, root := testBroker(t)
	resp := b.Handle(context.Background(), wire.Request{
		ID: "mkdir-deny", Subject: "alice", Tool: "file.mkdir",
		Resource: filepath.Join(filepath.Dir(root), "outside"),
		InvocationID: "mkdir-deny-op",
	})
	if resp.OK {
		t.Fatalf("outside mkdir allowed: %+v", resp)
	}
}

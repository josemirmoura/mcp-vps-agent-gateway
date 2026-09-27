package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sandbox"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
)

type fakeRunner struct {
	cancelled bool
}

func (f *fakeRunner) Start(_ context.Context, s sandbox.Spec) (Job, error) {
	return Job{ID: s.Unit, Unit: s.Unit, State: "running", Deadline: time.Now().Add(s.Runtime)}, nil
}
func (f *fakeRunner) Status(context.Context, Job) (string, error)    { return "running", nil }
func (f *fakeRunner) Tail(context.Context, Job, int) (string, error) { return "hello", nil }
func (f *fakeRunner) Cancel(context.Context, Job) error              { f.cancelled = true; return nil }

func TestManagerPersistsAndScopesJobs(t *testing.T) {
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	runner := &fakeRunner{}
	m := &Manager{State: store, Runner: runner}
	ctx := context.Background()

	rec, err := m.Start(ctx, "alice", "shell.exec", "/srv/app", "", sandbox.Spec{
		Unit: "job-1", Command: "echo hi", CWD: "/srv/app", Runtime: time.Minute, NetworkMode: "blocked",
	})
	if err != nil || rec.State != "running" {
		t.Fatalf("rec=%+v err=%v", rec, err)
	}
	if _, err := m.Status(ctx, "bob", rec.ID); err == nil {
		t.Fatal("different subject accessed job")
	}
	tail, err := m.Tail(ctx, "alice", rec.ID, 10)
	if err != nil || tail != "hello" {
		t.Fatalf("tail=%q err=%v", tail, err)
	}
	if err := m.Cancel(ctx, "alice", rec.ID); err != nil || !runner.cancelled {
		t.Fatalf("cancel err=%v cancelled=%v", err, runner.cancelled)
	}
}

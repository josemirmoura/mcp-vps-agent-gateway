package jobs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sandbox"
)

func TestSystemdRunnerSmoke(t *testing.T) {
	if os.Getenv("VPS_AGENT_SYSTEMD_JOB_SMOKE") != "1" {
		t.Skip("set VPS_AGENT_SYSTEMD_JOB_SMOKE=1 to run privileged systemd job smoke")
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run unavailable")
	}

	unit := fmt.Sprintf("vps-agent-ci-job-%d", os.Getpid())
	r := SystemdRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	job, err := r.Start(ctx, sandbox.Spec{
		Unit: unit, User: "vps-agent-exec",
		Command: "echo transient-job-proof; sleep 20",
		CWD: "/tmp", Runtime: 25 * time.Second,
		MemoryMaxBytes: 64 << 20, TasksMax: 16,
		NetworkMode: "blocked",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = exec.Command("systemctl", "stop", unit).Run()
		_ = exec.Command("systemctl", "reset-failed", unit).Run()
	})

	deadline := time.Now().Add(8 * time.Second)
	var status string
	for time.Now().Before(deadline) {
		status, err = r.Status(ctx, job)
		if err == nil && (status == "active" || status == "activating") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if status != "active" && status != "activating" {
		t.Fatalf("transient unit did not become active: status=%q err=%v", status, err)
	}

	var logs string
	for time.Now().Before(deadline) {
		logs, _ = r.Tail(ctx, job, 50)
		if strings.Contains(logs, "transient-job-proof") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(logs, "transient-job-proof") {
		t.Fatalf("expected command output in journal, got %q", logs)
	}

	if err := r.Cancel(ctx, job); err != nil {
		t.Fatal(err)
	}
}

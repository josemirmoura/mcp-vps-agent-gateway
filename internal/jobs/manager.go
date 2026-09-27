package jobs

import (
	"context"
	"errors"
	"fmt"
		"strings"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/hostexec"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sandbox"
)

type Job struct {
	ID       string    `json:"id"`
	Unit     string    `json:"unit"`
	State    string    `json:"state"`
	Deadline time.Time `json:"deadline"`
}

type Runner interface {
	Start(context.Context, sandbox.Spec) (Job, error)
	Status(context.Context, Job) (string, error)
	Tail(context.Context, Job, int) (string, error)
	Cancel(context.Context, Job) error
}

type SystemdRunner struct{}

func (SystemdRunner) Start(ctx context.Context, spec sandbox.Spec) (Job, error) {
	args, err := sandbox.BuildSystemdRunArgs(spec)
	if err != nil {
		return Job{}, err
	}
	out, err := hostexec.CommandContext(ctx, "systemd-run", args...).CombinedOutput()
	if err != nil {
		return Job{}, fmt.Errorf("systemd-run: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return Job{
		ID: spec.Unit, Unit: spec.Unit, State: "running",
		Deadline: time.Now().Add(spec.Runtime),
	}, nil
}

func (SystemdRunner) Status(ctx context.Context, job Job) (string, error) {
	out, err := hostexec.CommandContext(ctx, "systemctl", "show", job.Unit, "--property=ActiveState", "--value").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("systemctl show: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (SystemdRunner) Tail(ctx context.Context, job Job, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	if lines > 1000 {
		lines = 1000
	}
	out, err := hostexec.CommandContext(ctx, "journalctl", "-u", job.Unit, "-n", fmt.Sprint(lines), "--no-pager").CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), nil
}

func (SystemdRunner) Cancel(ctx context.Context, job Job) error {
	if job.Unit == "" {
		return errors.New("unit is required")
	}
	out, err := hostexec.CommandContext(ctx, "systemctl", "stop", job.Unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl stop: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

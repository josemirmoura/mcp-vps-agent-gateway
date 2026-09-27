package broker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ServiceManager interface {
	Status(context.Context, string) (string, error)
	Restart(context.Context, string) (string, error)
}

type SystemdManager struct{}

func (SystemdManager) Status(ctx context.Context, name string) (string, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "is-active", name).CombinedOutput()
	status := strings.TrimSpace(string(out))
	if err != nil {
		return status, fmt.Errorf("systemctl is-active %s: %w", name, err)
	}
	return status, nil
}

func (SystemdManager) Restart(ctx context.Context, name string) (string, error) {
	if out, err := exec.CommandContext(ctx, "systemctl", "restart", name).CombinedOutput(); err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("systemctl restart %s: %w", name, err)
	}
	return SystemdManager{}.Status(ctx, name)
}

package broker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type ServiceInfo struct {
	Name        string `json:"name"`
	Load        string `json:"load,omitempty"`
	Active      string `json:"active,omitempty"`
	Sub         string `json:"sub,omitempty"`
	Description string `json:"description,omitempty"`
}

type ServiceManager interface {
	List(context.Context) ([]ServiceInfo, error)
	Status(context.Context, string) (string, error)
	Logs(context.Context, string, int) (string, error)
	Start(context.Context, string) (string, error)
	Stop(context.Context, string) (string, error)
	Restart(context.Context, string) (string, error)
	Reload(context.Context, string) (string, error)
	Enable(context.Context, string) (string, error)
	Disable(context.Context, string) (string, error)
}

type SystemdManager struct{}

func (SystemdManager) List(ctx context.Context) ([]ServiceInfo, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-legend", "--no-pager", "--plain").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units: %s: %w", strings.TrimSpace(string(out)), err)
	}
	var result []ServiceInfo
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		desc := ""
		if len(fields) > 4 {
			desc = strings.Join(fields[4:], " ")
		}
		result = append(result, ServiceInfo{
			Name: fields[0], Load: fields[1], Active: fields[2], Sub: fields[3], Description: desc,
		})
	}
	return result, nil
}

func (SystemdManager) Status(ctx context.Context, name string) (string, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "is-active", name).CombinedOutput()
	status := strings.TrimSpace(string(out))
	if err != nil {
		return status, fmt.Errorf("systemctl is-active %s: %w", name, err)
	}
	return status, nil
}

func (SystemdManager) Logs(ctx context.Context, name string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	if lines > 1000 {
		lines = 1000
	}
	out, err := exec.CommandContext(ctx, "journalctl", "-u", name, "-n", strconv.Itoa(lines), "--no-pager", "-o", "short-iso").CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("journalctl %s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func systemctlAction(ctx context.Context, action, name string) (string, error) {
	out, err := exec.CommandContext(ctx, "systemctl", action, name).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("systemctl %s %s: %w", action, name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (SystemdManager) Start(ctx context.Context, name string) (string, error) {
	if _, err := systemctlAction(ctx, "start", name); err != nil {
		return "", err
	}
	return SystemdManager{}.Status(ctx, name)
}

func (SystemdManager) Stop(ctx context.Context, name string) (string, error) {
	if _, err := systemctlAction(ctx, "stop", name); err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, "systemctl", "is-active", name).CombinedOutput()
	status := strings.TrimSpace(string(out))
	if status == "inactive" || status == "failed" {
		return status, nil
	}
	if err != nil {
		return status, fmt.Errorf("systemctl is-active %s: %w", name, err)
	}
	return status, nil
}

func (SystemdManager) Restart(ctx context.Context, name string) (string, error) {
	if _, err := systemctlAction(ctx, "restart", name); err != nil {
		return "", err
	}
	return SystemdManager{}.Status(ctx, name)
}

func (SystemdManager) Reload(ctx context.Context, name string) (string, error) {
	if _, err := systemctlAction(ctx, "reload", name); err != nil {
		return "", err
	}
	return SystemdManager{}.Status(ctx, name)
}

func (SystemdManager) Enable(ctx context.Context, name string) (string, error) {
	if _, err := systemctlAction(ctx, "enable", name); err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, "systemctl", "is-enabled", name).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("systemctl is-enabled %s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (SystemdManager) Disable(ctx context.Context, name string) (string, error) {
	if _, err := systemctlAction(ctx, "disable", name); err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, "systemctl", "is-enabled", name).CombinedOutput()
	status := strings.TrimSpace(string(out))
	if status == "disabled" || status == "static" {
		return status, nil
	}
	if err != nil {
		return status, fmt.Errorf("systemctl is-enabled %s: %w", name, err)
	}
	return status, nil
}

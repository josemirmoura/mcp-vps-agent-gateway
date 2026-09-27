package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type DockerManager interface {
	Inspect(context.Context, string) (map[string]any, error)
	Logs(context.Context, string, int) (string, error)
	Restart(context.Context, string) (map[string]any, error)
}

type DockerCLI struct{}

func (DockerCLI) Inspect(ctx context.Context, name string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", name).Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect %s: %w", name, err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("docker inspect returned %d objects", len(rows))
	}
	return rows[0], nil
}

func (DockerCLI) Logs(ctx context.Context, name string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	if lines > 1000 {
		lines = 1000
	}
	out, err := exec.CommandContext(ctx, "docker", "logs", "--tail", strconv.Itoa(lines), name).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("docker logs %s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d DockerCLI) Restart(ctx context.Context, name string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "docker", "restart", name).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker restart %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return d.Inspect(ctx, name)
}

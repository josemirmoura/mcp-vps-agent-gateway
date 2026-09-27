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

type dockerInspectRaw struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	RestartCount int    `json:"RestartCount"`
	Config       struct {
		Image string `json:"Image"`
	} `json:"Config"`
	State struct {
		Status     string `json:"Status"`
		Running    bool   `json:"Running"`
		ExitCode   int    `json:"ExitCode"`
		Error      string `json:"Error"`
		StartedAt  string `json:"StartedAt"`
		FinishedAt string `json:"FinishedAt"`
		Health     *struct {
			Status        string `json:"Status"`
			FailingStreak int    `json:"FailingStreak"`
		} `json:"Health"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func (DockerCLI) Inspect(ctx context.Context, name string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", name).Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect %s: %w", name, err)
	}
	var rows []dockerInspectRaw
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("docker inspect returned %d objects", len(rows))
	}
	row := rows[0]

	networks := make(map[string]string, len(row.NetworkSettings.Networks))
	for network, cfg := range row.NetworkSettings.Networks {
		networks[network] = cfg.IPAddress
	}

	ports := make(map[string][]map[string]string, len(row.NetworkSettings.Ports))
	for containerPort, bindings := range row.NetworkSettings.Ports {
		safe := make([]map[string]string, 0, len(bindings))
		for _, b := range bindings {
			safe = append(safe, map[string]string{
				"host_ip": b.HostIP, "host_port": b.HostPort,
			})
		}
		ports[containerPort] = safe
	}

	state := map[string]any{
		"status":      row.State.Status,
		"running":     row.State.Running,
		"exit_code":   row.State.ExitCode,
		"error":       row.State.Error,
		"started_at":  row.State.StartedAt,
		"finished_at": row.State.FinishedAt,
	}
	if row.State.Health != nil {
		state["health"] = map[string]any{
			"status": row.State.Health.Status,
			"failing_streak": row.State.Health.FailingStreak,
		}
	}

	// Intentionally omit Config.Env, labels, mounts, host config and raw inspect
	// payload. Those fields routinely contain credentials or sensitive paths.
	return map[string]any{
		"id":            row.ID,
		"name":          strings.TrimPrefix(row.Name, "/"),
		"image":         row.Config.Image,
		"restart_count": row.RestartCount,
		"state":         state,
		"networks":      networks,
		"ports":         ports,
	}, nil
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

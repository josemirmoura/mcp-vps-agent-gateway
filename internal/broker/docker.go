package broker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const dockerOutputLimit = 1 << 20

type DockerManager interface {
	List(context.Context) ([]map[string]any, error)
	Inspect(context.Context, string) (map[string]any, error)
	Logs(context.Context, string, int) (string, error)
	Start(context.Context, string) (map[string]any, error)
	Stop(context.Context, string) (map[string]any, error)
	Restart(context.Context, string) (map[string]any, error)
	ComposeValidate(context.Context, string) error
	ComposePull(context.Context, string) (string, error)
	ComposeUp(context.Context, string) (string, error)
	ComposeDown(context.Context, string) (string, error)
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

func boundedOutput(out []byte) string {
	if len(out) > dockerOutputLimit {
		out = out[len(out)-dockerOutputLimit:]
		return "[truncated]\n" + strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func (DockerCLI) List(ctx context.Context) ([]map[string]any, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}
	var result []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Buffer(make([]byte, 64*1024), dockerOutputLimit)
	for scanner.Scan() {
		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("decode docker ps: %w", err)
		}
		// Docker ps formatted fields intentionally omit container environment.
		result = append(result, row)
	}
	return result, scanner.Err()
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
			safe = append(safe, map[string]string{"host_ip": b.HostIP, "host_port": b.HostPort})
		}
		ports[containerPort] = safe
	}
	state := map[string]any{
		"status": row.State.Status, "running": row.State.Running, "exit_code": row.State.ExitCode,
		"error": row.State.Error, "started_at": row.State.StartedAt, "finished_at": row.State.FinishedAt,
	}
	if row.State.Health != nil {
		state["health"] = map[string]any{"status": row.State.Health.Status, "failing_streak": row.State.Health.FailingStreak}
	}
	return map[string]any{
		"id": row.ID, "name": strings.TrimPrefix(row.Name, "/"), "image": row.Config.Image,
		"restart_count": row.RestartCount, "state": state, "networks": networks, "ports": ports,
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
		return boundedOutput(out), fmt.Errorf("docker logs %s: %w", name, err)
	}
	return boundedOutput(out), nil
}

func (d DockerCLI) Start(ctx context.Context, name string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "docker", "start", name).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker start %s: %s: %w", name, boundedOutput(out), err)
	}
	return d.Inspect(ctx, name)
}

func (d DockerCLI) Stop(ctx context.Context, name string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "docker", "stop", name).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker stop %s: %s: %w", name, boundedOutput(out), err)
	}
	return d.Inspect(ctx, name)
}

func (d DockerCLI) Restart(ctx context.Context, name string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "docker", "restart", name).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker restart %s: %s: %w", name, boundedOutput(out), err)
	}
	return d.Inspect(ctx, name)
}

func canonicalComposeDir(dir string) (string, error) {
	if !filepath.IsAbs(dir) {
		return "", errors.New("compose project directory must be absolute")
	}
	clean := filepath.Clean(dir)
	info, err := os.Lstat(clean)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("compose project directory must be a real directory, not a symlink")
	}
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	if resolved != clean {
		return "", errors.New("compose project directory resolves through symlink")
	}
	return clean, nil
}

func composeCommand(ctx context.Context, dir string, args ...string) (string, error) {
	dir, err := canonicalComposeDir(dir)
	if err != nil {
		return "", err
	}
	base := []string{"compose", "--project-directory", dir}
	cmd := exec.CommandContext(ctx, "docker", append(base, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return boundedOutput(out), fmt.Errorf("docker compose %s: %w", strings.Join(args, " "), err)
	}
	return boundedOutput(out), nil
}

func (DockerCLI) ComposeValidate(ctx context.Context, dir string) error {
	_, err := composeCommand(ctx, dir, "config", "-q")
	return err
}

func (DockerCLI) ComposePull(ctx context.Context, dir string) (string, error) {
	return composeCommand(ctx, dir, "pull")
}

func (DockerCLI) ComposeUp(ctx context.Context, dir string) (string, error) {
	return composeCommand(ctx, dir, "up", "-d", "--remove-orphans")
}

func (DockerCLI) ComposeDown(ctx context.Context, dir string) (string, error) {
	return composeCommand(ctx, dir, "down")
}

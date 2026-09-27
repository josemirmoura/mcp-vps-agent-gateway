package broker

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestDockerCLISmoke(t *testing.T) {
	if os.Getenv("VPS_AGENT_DOCKER_SMOKE") != "1" {
		t.Skip("set VPS_AGENT_DOCKER_SMOKE=1 to run Docker integration smoke")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const name = "vps-agent-ci-smoke"
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })

	if out, err := exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, "alpine:3.22", "sh", "-c", "echo ready; sleep 60").CombinedOutput(); err != nil {
		t.Fatalf("docker run: %s: %v", out, err)
	}

	d := DockerCLI{}
	info, err := d.Inspect(ctx, name)
	if err != nil || info["Name"] != "/"+name {
		t.Fatalf("inspect name=%v err=%v", info["Name"], err)
	}
	logs, err := d.Logs(ctx, name, 20)
	if err != nil || logs != "ready" {
		t.Fatalf("logs=%q err=%v", logs, err)
	}
	if _, err := d.Restart(ctx, name); err != nil {
		t.Fatal(err)
	}
}

package broker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"encoding/json"
	"strings"
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

	if out, err := exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, "-e", "PROOF_SECRET=do-not-leak", "alpine:3.22", "sh", "-c", "echo ready; sleep 60").CombinedOutput(); err != nil {
		t.Fatalf("docker run: %s: %v", out, err)
	}

	d := DockerCLI{}
	list, err := d.List(ctx)
	if err != nil || len(list) == 0 {
		t.Fatalf("docker list err=%v rows=%d", err, len(list))
	}
	info, err := d.Inspect(ctx, name)
	if err != nil || info["name"] != name {
		t.Fatalf("inspect name=%v err=%v", info["name"], err)
	}
	rawInfo, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawInfo), "do-not-leak") || strings.Contains(string(rawInfo), "PROOF_SECRET") {
		t.Fatalf("sanitized inspect leaked container environment: %s", rawInfo)
	}
	logs, err := d.Logs(ctx, name, 20)
	if err != nil || logs != "ready" {
		t.Fatalf("logs=%q err=%v", logs, err)
	}
	if _, err := d.Restart(ctx, name); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Stop(ctx, name); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Start(ctx, name); err != nil {
		t.Fatal(err)
	}

	composeDir := filepath.Join(t.TempDir(), "compose")
	if err := os.MkdirAll(composeDir, 0o750); err != nil {
		t.Fatal(err)
	}
	compose := "services:\n  app:\n    image: alpine:3.22\n    command: [\"sh\",\"-c\",\"echo compose-ready; sleep 30\"]\n"
	if err := os.WriteFile(filepath.Join(composeDir, "compose.yaml"), []byte(compose), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := d.ComposeValidate(ctx, composeDir); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ComposeUp(ctx, composeDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = d.ComposeDown(context.Background(), composeDir) })
	if _, err := d.ComposeDown(ctx, composeDir); err != nil {
		t.Fatal(err)
	}
}

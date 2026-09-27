package hostexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const envHostRoot = "VPS_AGENT_HOST_ROOT"

func Root() string {
	root := strings.TrimSpace(os.Getenv(envHostRoot))
	if root == "" || root == "/" {
		return ""
	}
	if !filepath.IsAbs(root) {
		return ""
	}
	return filepath.Clean(root)
}

func Path(canonical string) string {
	root := Root()
	if root == "" {
		return canonical
	}
	clean := filepath.Clean(canonical)
	if clean == "/" {
		return root
	}
	return filepath.Join(root, strings.TrimPrefix(clean, string(filepath.Separator)))
}

func commandArgs(name string, args ...string) (string, []string) {
	root := Root()
	if root == "" {
		return name, args
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("VPS_AGENT_HOST_NSENTER")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("VPS_AGENT_HOST_NSENTER")), "true") {
		nsArgs := []string{
			"--target", "1",
			"--mount", "--uts", "--ipc", "--net", "--pid", "--cgroup",
			"--root=/proc/1/root",
			"--wdns=/",
			"--",
			name,
		}
		nsArgs = append(nsArgs, args...)
		return "nsenter", nsArgs
	}
	chrootArgs := []string{root, name}
	chrootArgs = append(chrootArgs, args...)
	return "chroot", chrootArgs
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmdName, cmdArgs := commandArgs(name, args...)
	// #nosec G204 -- cmdName is selected by commandArgs from typed Broker operations; raw MCP shell input is not used as the executable name here.\n\treturn exec.CommandContext(ctx, cmdName, cmdArgs...)
}

func Command(name string, args ...string) *exec.Cmd {
	cmdName, cmdArgs := commandArgs(name, args...)
	// #nosec G204 -- cmdName is selected by commandArgs from typed Broker operations; raw MCP shell input is not used as the executable name here.\n\treturn exec.Command(cmdName, cmdArgs...)
}

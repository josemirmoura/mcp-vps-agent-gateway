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
	chrootArgs := []string{root, name}
	chrootArgs = append(chrootArgs, args...)
	return "chroot", chrootArgs
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmdName, cmdArgs := commandArgs(name, args...)
	return exec.CommandContext(ctx, cmdName, cmdArgs...)
}

func Command(name string, args ...string) *exec.Cmd {
	cmdName, cmdArgs := commandArgs(name, args...)
	return exec.Command(cmdName, cmdArgs...)
}

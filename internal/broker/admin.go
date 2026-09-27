package broker

// Typed host administration intentionally avoids generic shell construction.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
		"regexp"
	"strconv"
	"strings"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/hostexec"
)

var packageNameRE = regexp.MustCompile("^[a-z0-9][a-z0-9+.-]*(?::[a-z0-9]+)?$")
var userNameRE = regexp.MustCompile("^[a-z_][a-z0-9_-]{0,31}$")
var groupNameRE = regexp.MustCompile("^[a-z_][a-z0-9_-]{0,31}$")

func packageList(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	out, err := hostexec.CommandContext(ctx, "dpkg-query", "-W", "-f=${binary:Package}\\t${Version}\\t${db:Status-Status}\\n").Output()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() && len(result) < limit {
		p := strings.SplitN(s.Text(), "\t", 3)
		if len(p) != 3 {
			continue
		}
		result = append(result, map[string]any{"name": p[0], "version": p[1], "status": p[2]})
	}
	return result, s.Err()
}

func aptAction(ctx context.Context, action, name string) (string, error) {
	var args []string
	switch action {
	case "update":
		args = []string{"update"}
	case "install", "remove":
		if !packageNameRE.MatchString(name) {
			return "", errors.New("invalid package name")
		}
		args = []string{action, "-y", "--", name}
	default:
		return "", fmt.Errorf("unsupported package action %q", action)
	}
	cmd := hostexec.CommandContext(ctx, "apt-get", args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput()
	text := boundedOutput(out)
	if err != nil {
		return text, fmt.Errorf("apt-get %s: %w", action, err)
	}
	return text, nil
}

func userList() ([]map[string]any, error) {
	f, err := os.Open(hostexec.Path("/etc/passwd"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var result []map[string]any
	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.Split(s.Text(), ":")
		if len(p) < 7 {
			continue
		}
		uid, _ := strconv.Atoi(p[2])
		gid, _ := strconv.Atoi(p[3])
		result = append(result, map[string]any{"name": p[0], "uid": uid, "gid": gid, "home": p[5], "shell": p[6]})
	}
	return result, s.Err()
}

func userInspect(name string) (map[string]any, error) {
	if !userNameRE.MatchString(name) {
		return nil, errors.New("invalid username")
	}
	rows, err := userList()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["name"] == name {
			return row, nil
		}
	}
	return nil, os.ErrNotExist
}

func userAction(ctx context.Context, action, name string, createHome bool) (string, error) {
	if !userNameRE.MatchString(name) {
		return "", errors.New("invalid username")
	}
	switch action {
	case "add":
		args := []string{}
		if createHome {
			args = append(args, "-m")
		}
		args = append(args, name)
		return runBounded(ctx, "useradd", args...)
	case "delete":
		return runBounded(ctx, "userdel", name)
	case "lock":
		return runBounded(ctx, "usermod", "-L", name)
	case "unlock":
		return runBounded(ctx, "usermod", "-U", name)
	default:
		return "", fmt.Errorf("unsupported user action %q", action)
	}
}

func groupList() ([]map[string]any, error) {
	f, err := os.Open(hostexec.Path("/etc/group"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var result []map[string]any
	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.Split(s.Text(), ":")
		if len(p) < 4 {
			continue
		}
		gid, _ := strconv.Atoi(p[2])
		members := []string{}
		if p[3] != "" {
			members = strings.Split(p[3], ",")
		}
		result = append(result, map[string]any{"name": p[0], "gid": gid, "members": members})
	}
	return result, s.Err()
}

func groupInspect(name string) (map[string]any, error) {
	if !groupNameRE.MatchString(name) {
		return nil, errors.New("invalid group name")
	}
	rows, err := groupList()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["name"] == name {
			return row, nil
		}
	}
	return nil, os.ErrNotExist
}

func groupAction(ctx context.Context, action, name string) (string, error) {
	if !groupNameRE.MatchString(name) {
		return "", errors.New("invalid group name")
	}
	switch action {
	case "add":
		return runBounded(ctx, "groupadd", name)
	case "delete":
		return runBounded(ctx, "groupdel", name)
	default:
		return "", fmt.Errorf("unsupported group action %q", action)
	}
}

func firewallStatus(ctx context.Context) (string, error) {
	return runBounded(ctx, "ufw", "status", "verbose")
}

func firewallAction(ctx context.Context, action, port, protocol, source string) (string, error) {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("port must be 1..65535")
	}
	switch protocol {
	case "", "tcp", "udp":
	default:
		return "", errors.New("protocol must be tcp or udp")
	}
	if source != "" && strings.ContainsAny(source, " \t\r\n;|&$") {
		return "", errors.New("invalid source")
	}
	rule := port
	if protocol != "" {
		rule += "/" + protocol
	}
	var args []string
	switch action {
	case "allow", "deny":
		if source == "" {
			args = []string{action, rule}
		} else {
			args = []string{action, "from", source, "to", "any", "port", port}
			if protocol != "" {
				args = append(args, "proto", protocol)
			}
		}
	case "delete_allow", "delete_deny":
		base := strings.TrimPrefix(action, "delete_")
		args = []string{"delete", base}
		if source == "" {
			args = append(args, rule)
		} else {
			args = append(args, "from", source, "to", "any", "port", port)
			if protocol != "" {
				args = append(args, "proto", protocol)
			}
		}
	default:
		return "", fmt.Errorf("unsupported firewall action %q", action)
	}
	return runBounded(ctx, "ufw", args...)
}

func runBounded(ctx context.Context, name string, args ...string) (string, error) {
	out, err := hostexec.CommandContext(ctx, name, args...).CombinedOutput()
	text := boundedOutput(out)
	if err != nil {
		return text, fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

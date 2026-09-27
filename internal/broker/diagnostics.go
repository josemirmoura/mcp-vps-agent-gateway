package broker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
		"strconv"
	"strings"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/hostexec"
)

func diskInfo(ctx context.Context) ([]map[string]any, error) {
	out, err := hostexec.CommandContext(ctx, "df", "-P", "-B1").Output()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	s := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for s.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(s.Text())
		if len(fields) < 6 {
			continue
		}
		size, _ := strconv.ParseInt(fields[1], 10, 64)
		used, _ := strconv.ParseInt(fields[2], 10, 64)
		avail, _ := strconv.ParseInt(fields[3], 10, 64)
		result = append(result, map[string]any{
			"filesystem": fields[0], "bytes": size, "used_bytes": used,
			"available_bytes": avail, "use_percent": fields[4], "mountpoint": fields[5],
		})
	}
	return result, s.Err()
}

func memoryInfo() (map[string]any, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]any{}
	allow := map[string]bool{
		"MemTotal": true, "MemFree": true, "MemAvailable": true, "Buffers": true,
		"Cached": true, "SwapTotal": true, "SwapFree": true,
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		if !allow[key] {
			continue
		}
		kb, err := strconv.ParseInt(parts[1], 10, 64)
		if err == nil {
			out[key+"_bytes"] = kb * 1024
		}
	}
	return out, s.Err()
}

func processList(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	out, err := hostexec.CommandContext(ctx, "ps", "-eo", "pid=,user=,comm=,%cpu=,%mem=", "--sort=-%cpu").Output()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() && len(result) < limit {
		fields := strings.Fields(s.Text())
		if len(fields) < 5 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[len(fields)-2], 64)
		mem, _ := strconv.ParseFloat(fields[len(fields)-1], 64)
		result = append(result, map[string]any{
			"pid": pid, "user": fields[1], "command": strings.Join(fields[2:len(fields)-2], " "),
			"cpu_percent": cpu, "memory_percent": mem,
		})
	}
	return result, s.Err()
}

func processInspect(pid int) (map[string]any, error) {
	if pid <= 0 {
		return nil, errors.New("pid must be positive")
	}
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	allow := map[string]bool{
		"Name": true, "State": true, "Pid": true, "PPid": true, "Uid": true, "Gid": true,
		"Threads": true, "VmPeak": true, "VmSize": true, "VmRSS": true, "RssAnon": true, "RssFile": true,
	}
	out := map[string]any{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		key := line[:i]
		if !allow[key] {
			continue
		}
		out[key] = strings.TrimSpace(line[i+1:])
	}
	return out, s.Err()
}

func listenInfo(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	out, err := hostexec.CommandContext(ctx, "ss", "-lntupH").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ss: %s: %w", strings.TrimSpace(string(out)), err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > limit {
		lines = lines[:limit]
	}
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func networkCheck(ctx context.Context, destination string, timeout time.Duration) (map[string]any, error) {
	if destination == "" {
		return nil, errors.New("destination host:port is required")
	}
	if _, _, err := net.SplitHostPort(destination); err != nil {
		return nil, fmt.Errorf("destination must be host:port: %w", err)
	}
	if timeout <= 0 || timeout > 15*time.Second {
		timeout = 5 * time.Second
	}
	start := time.Now()
	conn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", destination)
	elapsed := time.Since(start)
	if err != nil {
		return map[string]any{"destination": destination, "reachable": false, "latency_ms": elapsed.Milliseconds(), "error": err.Error()}, nil
	}
	_ = conn.Close()
	return map[string]any{"destination": destination, "reachable": true, "latency_ms": elapsed.Milliseconds()}, nil
}

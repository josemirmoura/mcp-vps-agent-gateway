package sandbox

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Spec struct {
	Unit           string
	User           string
	Command        string
	CWD            string
	ReadOnlyPaths  []string
	ReadWritePaths []string
	InaccessiblePaths []string
	Runtime        time.Duration
	MemoryMaxBytes int64
	TasksMax       int
	NetworkMode    string
	Admin          bool
}

func BuildSystemdRunArgs(s Spec) ([]string, error) {
	if s.Unit == "" || s.Command == "" || s.CWD == "" {
		return nil, errors.New("unit, command and cwd are required")
	}
	if !filepath.IsAbs(s.CWD) {
		return nil, errors.New("cwd must be absolute")
	}
	if strings.ContainsAny(s.Unit, "/ \t\n") {
		return nil, errors.New("invalid unit name")
	}
	if s.Runtime <= 0 {
		s.Runtime = 5 * time.Minute
	}
	if s.MemoryMaxBytes <= 0 {
		s.MemoryMaxBytes = 512 << 20
	}
	if s.TasksMax <= 0 {
		s.TasksMax = 100
	}
	if s.User == "" {
		s.User = "vps-agent-exec"
	}

	args := []string{
		"--unit=" + s.Unit,
		"--collect",
		"--no-block",
		"--quiet",
		"--uid=" + s.User,
		"--working-directory=" + s.CWD,
		"--property=NoNewPrivileges=yes",
		"--property=PrivateTmp=yes",
		"--property=ProtectSystem=strict",
		"--property=ProtectHome=yes",
		"--property=ProtectProc=invisible",
		"--property=ProcSubset=pid",
		"--property=PrivateDevices=yes",
		"--property=ProtectKernelTunables=yes",
		"--property=ProtectControlGroups=yes",
		"--property=ProtectKernelModules=yes",
		"--property=RestrictSUIDSGID=yes",
		"--property=LockPersonality=yes",
		"--property=MemoryMax=" + strconv.FormatInt(s.MemoryMaxBytes, 10),
		"--property=TasksMax=" + strconv.Itoa(s.TasksMax),
		"--property=RuntimeMaxSec=" + strconv.FormatInt(int64(s.Runtime.Seconds()), 10),
	}
	for _, p := range s.InaccessiblePaths {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("inaccessible path must be absolute: %q", p)
		}
		args = append(args, "--property=InaccessiblePaths="+p)
	}
	for _, p := range s.ReadOnlyPaths {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("read-only path must be absolute: %q", p)
		}
		args = append(args, "--property=ReadOnlyPaths="+p)
	}
	for _, p := range s.ReadWritePaths {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("read-write path must be absolute: %q", p)
		}
		args = append(args, "--property=ReadWritePaths="+p)
	}
	switch s.NetworkMode {
	case "", "blocked":
		args = append(args, "--property=PrivateNetwork=yes")
	case "unrestricted":
		// Deliberate explicit policy choice.
	case "allowlist":
		return nil, errors.New("hostname/IP egress allowlist is not implemented; fail closed")
	default:
		return nil, fmt.Errorf("unsupported network mode %q", s.NetworkMode)
	}
	args = append(args, "--", "/bin/sh", "-c", s.Command)
	return args, nil
}

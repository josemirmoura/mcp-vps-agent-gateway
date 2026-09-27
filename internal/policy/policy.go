package policy

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int      `yaml:"version"`
	Mode     string   `yaml:"mode"`
	Enabled  bool     `yaml:"enabled,omitempty"`
	Features Features `yaml:"features,omitempty"`

	Filesystem   FilesystemPolicy `yaml:"filesystem"`
	Network      NetworkPolicy    `yaml:"network"`
	Services     ResourcePolicy   `yaml:"services"`
	Docker       ResourcePolicy   `yaml:"docker"`
	Compose      ResourcePolicy   `yaml:"compose,omitempty"`
	Packages     ResourcePolicy   `yaml:"packages,omitempty"`
	Users        ResourcePolicy   `yaml:"users,omitempty"`
	Groups       ResourcePolicy   `yaml:"groups,omitempty"`
	Firewall     ResourcePolicy   `yaml:"firewall,omitempty"`
	Diagnostics  []string         `yaml:"diagnostics,omitempty"`
	Shell        ShellPolicy      `yaml:"shell"`
	Privilege    PrivilegePolicy  `yaml:"privilege"`
	Replay       ReplayPolicy     `yaml:"replay"`
	Grant        GrantPolicy      `yaml:"grant,omitempty"`
	Capabilities []string         `yaml:"capabilities,omitempty"`
}

type Features struct {
	FullModeEnabled bool `yaml:"full_mode_enabled"`
}

type FilesystemPolicy struct {
	Read    []string `yaml:"read"`
	Write   []string `yaml:"write"`
	Actions []string `yaml:"actions,omitempty"`
}

type NetworkPolicy struct {
	Mode                                 string   `yaml:"mode"`
	Destinations                         []string `yaml:"destinations"`
	UnrestrictedRequiresSeparateApproval bool     `yaml:"unrestricted_requires_separate_approval,omitempty"`
}

type ResourcePolicy struct {
	Inspect []string `yaml:"inspect"`
	Manage  []string `yaml:"manage"`
	Actions []string `yaml:"actions"`
}

type ShellPolicy struct {
	Enabled           bool     `yaml:"enabled"`
	RunAs             string   `yaml:"run_as,omitempty"`
	CWDRoots          []string `yaml:"cwd_roots"`
	MaxRuntimeSeconds int      `yaml:"max_runtime_seconds"`
	MaxOutputBytes    int      `yaml:"max_output_bytes"`
	MaxMemoryBytes    int64    `yaml:"max_memory_bytes,omitempty"`
	MaxTasks          int      `yaml:"max_tasks,omitempty"`
	AllowHostRead     bool     `yaml:"allow_host_read,omitempty"`
}

type PrivilegePolicy struct {
	Admin string `yaml:"admin"`
}

type ReplayPolicy struct {
	RequireIdempotencyForSafeWrites bool `yaml:"require_idempotency_for_safe_writes"`
	BlindRetryNonReplaySafe         bool `yaml:"blind_retry_non_replay_safe"`
}

type GrantPolicy struct {
	Required          bool `yaml:"required"`
	MaxTTLMinutes     int  `yaml:"max_ttl_minutes"`
	OutOfBandApproval bool `yaml:"out_of_band_approval,omitempty"`
	StepUpAuth        bool `yaml:"step_up_auth,omitempty"`
}

func Load(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode policy: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported policy version %d", c.Version)
	}
	switch c.Mode {
	case "controlled", "scoped", "full":
	default:
		return fmt.Errorf("invalid mode %q", c.Mode)
	}
	switch c.Network.Mode {
	case "", "blocked", "allowlist", "unrestricted":
	default:
		return fmt.Errorf("invalid network mode %q", c.Network.Mode)
	}
	switch c.Privilege.Admin {
	case "", "deny", "broker-only", "allow":
	default:
		return fmt.Errorf("invalid privilege.admin %q", c.Privilege.Admin)
	}
	if c.Replay.BlindRetryNonReplaySafe {
		return errors.New("blind_retry_non_replay_safe must be false")
	}
	if c.Mode == "full" && c.Enabled && !c.Features.FullModeEnabled {
		return errors.New("full policy cannot be enabled while features.full_mode_enabled is false")
	}
	if c.Grant.MaxTTLMinutes < 0 {
		return errors.New("grant.max_ttl_minutes cannot be negative")
	}
	return nil
}

func matchAny(patterns []string, value string) bool {
	for _, p := range patterns {
		if p == "*" || p == value {
			return true
		}
		ok, err := path.Match(p, value)
		if err == nil && ok {
			return true
		}
	}
	return false
}


func containsAction(actions []string, action string) bool {
	action = NormalizeCapability(action)
	for _, allowed := range actions {
		allowed = NormalizeCapability(allowed)
		if allowed == "*" || allowed == action {
			return true
		}
	}
	return false
}

func (c *Config) CanFilesystem(action string) bool {
	if len(c.Filesystem.Actions) == 0 {
		switch NormalizeCapability(action) {
		case "read", "write", "mkdir":
			return true
		default:
			return false
		}
	}
	return containsAction(c.Filesystem.Actions, action)
}

func (c *Config) CanCompose(resource, action string) bool {
	if !matchAny(c.Compose.Actions, action) {
		return false
	}
	if action == "inspect" || action == "validate" {
		return matchAny(c.Compose.Inspect, resource) || matchAny(c.Compose.Manage, resource)
	}
	return matchAny(c.Compose.Manage, resource)
}

func (c *Config) CanPackage(name, action string) bool {
	if !matchAny(c.Packages.Actions, action) {
		return false
	}
	if action == "update" {
		return true
	}
	if action == "list" || action == "status" {
		return matchAny(c.Packages.Inspect, name) || matchAny(c.Packages.Manage, name)
	}
	return matchAny(c.Packages.Manage, name)
}

func (c *Config) CanUser(name, action string) bool {
	if !matchAny(c.Users.Actions, action) {
		return false
	}
	if action == "list" || action == "inspect" {
		return matchAny(c.Users.Inspect, name) || matchAny(c.Users.Manage, name)
	}
	return matchAny(c.Users.Manage, name)
}


func (c *Config) CanGroup(name, action string) bool {
	if !matchAny(c.Groups.Actions, action) {
		return false
	}
	if action == "list" || action == "inspect" {
		return matchAny(c.Groups.Inspect, name) || matchAny(c.Groups.Manage, name)
	}
	return matchAny(c.Groups.Manage, name)
}

func (c *Config) CanFirewall(action string) bool {
	return matchAny(c.Firewall.Actions, action)
}

func (c *Config) CanDiagnostic(action string) bool {
	return containsAction(c.Diagnostics, action)
}

func withinAnyRoot(roots []string, target string) bool {
	if !filepath.IsAbs(target) {
		return false
	}
	target = filepath.Clean(target)
	for _, root := range roots {
		root = filepath.Clean(root)
		rel, err := filepath.Rel(root, target)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

func (c *Config) CanShellCWD(cwd string) bool {
	return c.Shell.Enabled && withinAnyRoot(c.Shell.CWDRoots, cwd)
}

func (c *Config) CanNetworkDestination(destination string) bool {
	switch c.Network.Mode {
	case "unrestricted":
		return true
	case "allowlist":
		return matchAny(c.Network.Destinations, destination)
	default:
		return false
	}
}

func (c *Config) CanService(name, action string) bool {
	if !matchAny(c.Services.Actions, action) {
		return false
	}
	if action == "status" || action == "inspect" {
		return matchAny(c.Services.Inspect, name) || matchAny(c.Services.Manage, name)
	}
	return matchAny(c.Services.Manage, name)
}

func (c *Config) CanDocker(name, action string) bool {
	if !matchAny(c.Docker.Actions, action) {
		return false
	}
	if action == "inspect" || action == "logs" {
		return matchAny(c.Docker.Inspect, name) || matchAny(c.Docker.Manage, name)
	}
	return matchAny(c.Docker.Manage, name)
}

func (c *Config) FileRoots(write bool) []string {
	if write {
		return append([]string(nil), c.Filesystem.Write...)
	}
	return append([]string(nil), c.Filesystem.Read...)
}

func (c *Config) CanGrant(capability string) bool {
	if c.Mode != "full" || !c.Enabled || !c.Features.FullModeEnabled {
		return false
	}
	capability = NormalizeCapability(capability)
	for _, allowed := range c.Capabilities {
		if NormalizeCapability(allowed) == capability {
			return true
		}
	}
	return false
}


func (c *Config) ShellRuntimeLimit() time.Duration {
	seconds := c.Shell.MaxRuntimeSeconds
	if seconds <= 0 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func (c *Config) ShellMemoryLimit() int64 {
	if c.Shell.MaxMemoryBytes <= 0 {
		return 512 << 20
	}
	return c.Shell.MaxMemoryBytes
}

func (c *Config) ShellTasksLimit() int {
	if c.Shell.MaxTasks <= 0 {
		return 100
	}
	return c.Shell.MaxTasks
}

func (c *Config) ShellMayReadHost() bool {
	if c.Shell.AllowHostRead {
		return true
	}
	for _, root := range c.Filesystem.Read {
		if filepath.Clean(root) == string(filepath.Separator) {
			return true
		}
	}
	return false
}

func (c *Config) MaxGrantTTL() time.Duration {
	if c.Grant.MaxTTLMinutes <= 0 {
		return time.Hour
	}
	return time.Duration(c.Grant.MaxTTLMinutes) * time.Minute
}

func NormalizeCapability(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

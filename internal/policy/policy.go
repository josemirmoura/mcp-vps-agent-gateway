package policy

import (
	"errors"
	"fmt"
	"os"
	"path"
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
	Read  []string `yaml:"read"`
	Write []string `yaml:"write"`
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
	CWDRoots          []string `yaml:"cwd_roots"`
	MaxRuntimeSeconds int      `yaml:"max_runtime_seconds"`
	MaxOutputBytes    int      `yaml:"max_output_bytes"`
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

func (c *Config) MaxGrantTTL() time.Duration {
	if c.Grant.MaxTTLMinutes <= 0 {
		return time.Hour
	}
	return time.Duration(c.Grant.MaxTTLMinutes) * time.Minute
}

func NormalizeCapability(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

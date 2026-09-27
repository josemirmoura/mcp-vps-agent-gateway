package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

const version = "0.1.0-rc.1"

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	v = strings.TrimSpace(v)
	if v != "" {
		*m = append(*m, v)
	}
	return nil
}

type installState struct {
	Version string
	State string
	InstalledAt time.Time
	Endpoint string
	AuthMode string
	Subject string
	BaselineAudit int64
	VerifiedAt time.Time
	VerifiedSeq int64
	PolicyPath string
	TutorialShown bool
}

type auditStatusResult struct {
	Valid bool
	Events int64
	HeadHash string
}

type bearerRoundTripper struct{ token string }

func (b bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "install":
		err = install(os.Args[2:])
	case "upgrade":
		err = upgrade(os.Args[2:])
	case "verify-chatgpt":
		err = verifyChatGPT(os.Args[2:])
	case "status":
		err = showStatus()
	case "uninstall":
		err = uninstall(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: vps-agent-setup <install|upgrade|verify-chatgpt|status|uninstall|version>")
}

func requireRoot() error {
	if os.Geteuid() != 0 {
		return errors.New("this operation must run as root")
	}
	return nil
}

func csv(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func prompt(reader *bufio.Reader, label, def string) (string, error) {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

func install(args []string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	sourceDir := fs.String("source-dir", ".", "directory containing built binaries")
	var scopes, services, dockerResources, composeProjects, netDestinations multiFlag
	var packages, users, groups multiFlag
	fs.Var(&scopes, "scope", "filesystem root delegated to MCP; repeatable")
	fs.Var(&services, "service", "systemd unit delegated to MCP; repeatable")
	fs.Var(&dockerResources, "docker", "Docker resource delegated to MCP; repeatable")
	fs.Var(&composeProjects, "compose", "Docker Compose project directory; repeatable")
	fs.Var(&netDestinations, "network-destination", "allowed host:port; repeatable")
	fs.Var(&packages, "package", "APT package name or *; repeatable")
	fs.Var(&users, "user", "local username or *; repeatable")
	fs.Var(&groups, "group", "local group name or *; repeatable")
	fsActions := fs.String("fs-actions", "list,stat,read,mkdir,write,patch,copy,move,remove,hash", "filesystem actions")
	serviceActions := fs.String("service-actions", "status,logs,restart", "systemd actions")
	dockerActions := fs.String("docker-actions", "list,inspect,logs,restart", "Docker actions")
	composeActions := fs.String("compose-actions", "validate,pull,up,down", "Compose actions")
	diagnostics := fs.String("diagnostics", "system.disk,system.memory,process.list,process.inspect,network.listen", "diagnostics")
	packageActions := fs.String("package-actions", "", "package actions")
	userActions := fs.String("user-actions", "", "user actions")
	groupActions := fs.String("group-actions", "", "group actions")
	firewallActions := fs.String("firewall-actions", "", "firewall actions")
	shellEnabled := fs.Bool("shell", false, "enable scoped shell/jobs")
	shellUser := fs.String("shell-user", "vps-agent-exec", "Linux user for scoped shell/jobs")
	shellHostRead := fs.Bool("shell-host-read", false, "allow scoped shell to read entire host")
	networkMode := fs.String("network-mode", "blocked", "blocked, allowlist, unrestricted")
	listen := fs.String("listen", "127.0.0.1:8080", "Gateway listen address")
	endpoint := fs.String("endpoint-url", "", "final MCP endpoint shown in tutorial")
	authMode := fs.String("auth-mode", "static", "static, oidc, none")
	subject := fs.String("subject", "operator", "expected authenticated subject")
	oidcIssuer := fs.String("oidc-issuer", "", "OIDC issuer")
	oidcAudience := fs.String("oidc-audience", "", "OIDC audience/resource")
	requiredScopes := fs.String("required-scopes", "", "OIDC scopes")
	enableFull := fs.Bool("enable-full", false, "enable temporary capability elevation")
	grantCaps := fs.String("grant-capabilities", "shell.admin", "grantable capabilities")
	nonInteractive := fs.Bool("non-interactive", false, "disable prompts")
	yes := fs.Bool("yes", false, "confirm generated policy")
	force := fs.Bool("force", false, "replace existing configuration")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat("/etc/vps-agent/policy.yaml"); err == nil && !*force {
		return errors.New("existing installation found; use upgrade or --force")
	}

	if len(scopes) == 0 && !*nonInteractive {
		reader := bufio.NewReader(os.Stdin)
		raw, err := prompt(reader, "Filesystem roots delegated to MCP (comma-separated)", "/opt/my-app")
		if err != nil { return err }
		scopes = csv(raw)
		raw, err = prompt(reader, "Filesystem capabilities", *fsActions)
		if err != nil { return err }
		*fsActions = raw
		raw, err = prompt(reader, "Enable scoped shell? (yes/no)", "no")
		if err != nil { return err }
		*shellEnabled = strings.EqualFold(raw, "yes") || strings.EqualFold(raw, "y")
		raw, err = prompt(reader, "Authentication mode (static/oidc/none)", *authMode)
		if err != nil { return err }
		*authMode = raw
		raw, err = prompt(reader, "Expected MCP subject", *subject)
		if err != nil { return err }
		*subject = raw
	}
	if len(scopes) == 0 {
		return errors.New("at least one --scope is required")
	}

	var readRoots, writeRoots []string
	for _, root := range scopes {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("scope must be absolute: %s", root)
		}
		root = filepath.Clean(root)
		if root != "/" {
			if err := os.MkdirAll(root, 0750); err != nil {
				return fmt.Errorf("create scope %s: %w", root, err)
			}
		}
		readRoots = append(readRoots, root)
		writeRoots = append(writeRoots, root)
	}
	for _, dir := range composeProjects {
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("compose project must be absolute: %s", dir)
		}
	}

	cfg := policy.Config{
		Version: 1,
		Mode: "scoped",
		Filesystem: policy.FilesystemPolicy{Read: readRoots, Write: writeRoots, Actions: csv(*fsActions)},
		Network: policy.NetworkPolicy{Mode: *networkMode, Destinations: netDestinations},
		Services: policy.ResourcePolicy{Inspect: services, Manage: services, Actions: csv(*serviceActions)},
		Docker: policy.ResourcePolicy{Inspect: dockerResources, Manage: dockerResources, Actions: csv(*dockerActions)},
		Compose: policy.ResourcePolicy{Inspect: composeProjects, Manage: composeProjects, Actions: csv(*composeActions)},
		Packages: policy.ResourcePolicy{Inspect: packages, Manage: packages, Actions: csv(*packageActions)},
		Users: policy.ResourcePolicy{Inspect: users, Manage: users, Actions: csv(*userActions)},
		Groups: policy.ResourcePolicy{Inspect: groups, Manage: groups, Actions: csv(*groupActions)},
		Firewall: policy.ResourcePolicy{Actions: csv(*firewallActions)},
		Diagnostics: csv(*diagnostics),
		Shell: policy.ShellPolicy{
			Enabled: *shellEnabled, RunAs: *shellUser, CWDRoots: readRoots, MaxRuntimeSeconds: 900,
			MaxOutputBytes: 1 << 20, MaxMemoryBytes: 1 << 30, MaxTasks: 256,
			AllowHostRead: *shellHostRead,
		},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true, BlindRetryNonReplaySafe: false},
		Grant: policy.GrantPolicy{Required: true, MaxTTLMinutes: 60, OutOfBandApproval: true, StepUpAuth: true},
	}
	if *enableFull {
		cfg.Mode = "full"
		cfg.Enabled = true
		cfg.Features.FullModeEnabled = true
		cfg.Capabilities = csv(*grantCaps)
	}
	if err := cfg.Validate(); err != nil { return err }

	policyRaw, err := yaml.Marshal(&cfg)
	if err != nil { return err }
	fmt.Println("\nEffective MCP authority:")
	fmt.Println(string(policyRaw))
	if !*yes && !*nonInteractive {
		reader := bufio.NewReader(os.Stdin)
		answer, err := prompt(reader, "Activate this policy? (yes/no)", "no")
		if err != nil { return err }
		if !strings.EqualFold(answer, "yes") && !strings.EqualFold(answer, "y") {
			return errors.New("installation cancelled")
		}
	}
	if *nonInteractive && !*yes {
		return errors.New("--non-interactive requires --yes")
	}

	if err := provisionAccounts(); err != nil { return err }
	if err := installBinaries(*sourceDir); err != nil { return err }
	if err := os.MkdirAll("/etc/vps-agent", 0750); err != nil { return err }
	if err := os.MkdirAll("/var/lib/vps-agent", 0750); err != nil { return err }
	_ = os.Chown("/var/lib/vps-agent", 0, lookupGroupID("vps-agent"))
	if err := os.Chmod("/var/lib/vps-agent", 0750); err != nil { return err }
	if err := writeRootFile("/etc/vps-agent/policy.yaml", policyRaw, 0600); err != nil { return err }

	adminToken, err := randomToken()
	if err != nil { return err }
	staticToken := ""
	switch *authMode {
	case "static":
		staticToken, err = randomToken()
		if err != nil { return err }
	case "oidc":
		if *oidcIssuer == "" || *oidcAudience == "" {
			return errors.New("oidc auth requires --oidc-issuer and --oidc-audience")
		}
	case "none":
		host, _, err := net.SplitHostPort(*listen)
		if err != nil { return fmt.Errorf("invalid listen address: %w", err) }
		if host != "127.0.0.1" && host != "::1" && host != "localhost" {
			return errors.New("auth-mode none is allowed only on loopback")
		}
	default:
		return fmt.Errorf("unsupported auth mode %q", *authMode)
	}

	gatewayEnv := []string{
		"VPS_AGENT_BROKER_SOCKET=/run/vps-agent/broker.sock",
		"VPS_AGENT_LISTEN="+*listen,
		"VPS_AGENT_AUTH_MODE="+*authMode,
		"VPS_AGENT_STATIC_SUBJECT="+*subject,
	}
	if staticToken != "" { gatewayEnv = append(gatewayEnv, "VPS_AGENT_STATIC_TOKEN="+staticToken) }
	if *authMode == "oidc" {
		gatewayEnv = append(gatewayEnv,
			"VPS_AGENT_OIDC_ISSUER="+*oidcIssuer,
			"VPS_AGENT_OIDC_AUDIENCE="+*oidcAudience,
			"VPS_AGENT_REQUIRED_SCOPES="+*requiredScopes)
	}
	brokerEnv := []string{
		"VPS_AGENT_POLICY=/etc/vps-agent/policy.yaml",
		"VPS_AGENT_BROKER_SOCKET=/run/vps-agent/broker.sock",
		"VPS_AGENT_STATE_DB=/var/lib/vps-agent/state.db",
		"VPS_AGENT_ADMIN_TOKEN="+adminToken,
		"VPS_AGENT_EXPECTED_SUBJECT="+*subject,
	}
	if err := writeRootFile("/etc/vps-agent/gateway.env", []byte(strings.Join(gatewayEnv, "\n")+"\n"), 0600); err != nil { return err }
	if err := writeRootFile("/etc/vps-agent/broker.env", []byte(strings.Join(brokerEnv, "\n")+"\n"), 0600); err != nil { return err }

	if err := installUnits(writeRoots, len(cfg.Packages.Actions) > 0, len(cfg.Users.Actions) > 0 || len(cfg.Groups.Actions) > 0, len(cfg.Firewall.Actions) > 0); err != nil {
		return err
	}
	if err := systemctl("daemon-reload"); err != nil { return err }
	if err := systemctl("enable", "--now", "vps-agent-broker.service", "vps-agent-gateway.service"); err != nil { return err }
	if err := waitReady(*listen, 20*time.Second); err != nil { return err }
	if *authMode == "static" {
		if err := mcpSmoke(*listen, staticToken); err != nil { return fmt.Errorf("local MCP smoke: %w", err) }
	}
	status, err := brokerAuditStatus(adminToken)
	if err != nil { return err }
	if *endpoint == "" { *endpoint = localEndpoint(*listen) }

	st := installState{
		Version: version, State: "awaiting_chatgpt_verification", InstalledAt: time.Now().UTC(),
		Endpoint: *endpoint, AuthMode: *authMode, Subject: *subject, BaselineAudit: status.Events,
		PolicyPath: "/etc/vps-agent/policy.yaml", TutorialShown: true,
	}
	if err := writeJSON("/var/lib/vps-agent/install-status.json", st, 0600); err != nil { return err }

	fmt.Println("\nRuntime installation: OK")
	fmt.Println("Policy activation: OK")
	fmt.Println("Local readiness: OK")
	fmt.Println("Installation state: AWAITING_CHATGPT_VERIFICATION")
	fmt.Println("MCP endpoint:", st.Endpoint)
	fmt.Println("Expected subject:", st.Subject)
	if staticToken != "" { fmt.Println("Static lab token (store securely):", staticToken) }
	printChatGPTTutorial(st)
	fmt.Println("\nAfter connecting ChatGPT, ask it to call system.info, then run:")
	fmt.Println("  sudo vps-agent-setup verify-chatgpt --timeout 5m")
	fmt.Println("The installation is NOT complete until that command succeeds.")
	return nil
}

func brokerAuditStatus(token string) (auditStatusResult, error) {
	resp, err := brokerCall(token, wire.Request{ID: "setup-audit-status", Tool: "admin.audit.status"})
	if err != nil { return auditStatusResult{}, err }
	var out auditStatusResult
	if err := json.Unmarshal(resp.Result, &out); err != nil { return out, err }
	if !out.Valid { return out, errors.New("audit chain is invalid") }
	return out, nil
}

func brokerCall(token string, req wire.Request) (wire.Response, error) {
	req.AdminToken = token
	resp, err := (ipc.Client{Socket: "/run/vps-agent/broker.sock", Timeout: 10 * time.Second}).Call(context.Background(), req)
	if err != nil { return resp, err }
	if !resp.OK {
		if resp.Error != nil { return resp, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message) }
		return resp, errors.New("broker request failed")
	}
	return resp, nil
}

func localEndpoint(listen string) string {
	if strings.HasPrefix(listen, ":") { return "http://127.0.0.1"+listen+"/mcp" }
	return "http://"+listen+"/mcp"
}

func healthURL(listen string) string {
	if strings.HasPrefix(listen, ":") { return "http://127.0.0.1"+listen+"/healthz" }
	return "http://"+listen+"/healthz"
}

func waitReady(listen string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL(listen))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if _, err := os.Stat("/run/vps-agent/broker.sock"); err == nil { return nil }
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return errors.New("Gateway/Broker readiness timeout")
}

func mcpSmoke(listen, token string) error {
	client := mcp.NewClient(&mcp.Implementation{Name: "vps-agent-setup", Version: version}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: localEndpoint(listen)}
	if token != "" { transport.HTTPClient = &http.Client{Transport: bearerRoundTripper{token: token}} }
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil { return err }
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "system.info", Arguments: map[string]any{}})
	if err != nil { return err }
	if result.IsError { return errors.New("system.info returned tool error") }
	return nil
}

func provisionAccounts() error {
	if err := runIgnoreExists("groupadd", "--system", "vps-agent"); err != nil { return err }
	if err := runIgnoreExists("useradd", "--system", "--gid", "vps-agent", "--no-create-home", "--shell", "/usr/sbin/nologin", "vps-agent"); err != nil { return err }
	if err := runIgnoreExists("useradd", "--system", "--no-create-home", "--shell", "/usr/sbin/nologin", "vps-agent-exec"); err != nil { return err }
	return nil
}

func runIgnoreExists(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err == nil { return nil }
	text := strings.ToLower(string(out))
	if strings.Contains(text, "already exists") || strings.Contains(text, "already in use") { return nil }
	return fmt.Errorf("%s: %s: %w", name, strings.TrimSpace(string(out)), err)
}

func installBinaries(sourceDir string) error {
	for _, name := range []string{"vps-agent-gateway", "vps-agent-broker", "vps-agent", "vps-agent-setup"} {
		src := filepath.Join(sourceDir, name)
		if _, err := os.Stat(src); err != nil { return fmt.Errorf("missing install binary %s: %w", src, err) }
		if err := copyFile(src, filepath.Join("/usr/local/bin", name), 0755); err != nil { return err }
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	tmp := dst+".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil { return err }
	if _, err := io.Copy(out, in); err != nil { _ = out.Close(); _ = os.Remove(tmp); return err }
	if err := out.Sync(); err != nil { _ = out.Close(); return err }
	if err := out.Close(); err != nil { return err }
	if err := os.Chmod(tmp, mode); err != nil { return err }
	return os.Rename(tmp, dst)
}

func writeRootFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil { return err }
	tmp := path+".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil { return err }
	if err := os.Chmod(tmp, mode); err != nil { return err }
	if err := os.Chown(tmp, 0, 0); err != nil { return err }
	return os.Rename(tmp, path)
}

func lookupGroupID(name string) int {
	out, err := exec.Command("getent", "group", name).Output()
	if err != nil { return 0 }
	p := strings.Split(strings.TrimSpace(string(out)), ":")
	if len(p) < 3 { return 0 }
	gid, _ := strconv.Atoi(p[2])
	return gid
}

func installUnits(writeRoots []string, packages, identities, firewall bool) error {
	gatewayUnit := strings.Join([]string{
		"[Unit]", "Description=MCP VPS Agent Gateway", "After=network-online.target vps-agent-broker.service",
		"Wants=network-online.target", "Requires=vps-agent-broker.service", "", "[Service]", "Type=simple",
		"User=vps-agent", "Group=vps-agent", "EnvironmentFile=-/etc/vps-agent/gateway.env",
		"ExecStart=/usr/local/bin/vps-agent-gateway", "Restart=on-failure", "RestartSec=2",
		"NoNewPrivileges=yes", "PrivateTmp=yes", "ProtectSystem=strict", "ProtectHome=yes",
		"ReadWritePaths=/run/vps-agent", "RestrictSUIDSGID=yes", "LockPersonality=yes", "UMask=0077",
		"", "[Install]", "WantedBy=multi-user.target", "",
	}, "\n")
	brokerUnit := strings.Join([]string{
		"[Unit]", "Description=MCP VPS Agent Privileged Broker", "After=local-fs.target", "",
		"[Service]", "Type=simple", "User=root", "Group=vps-agent",
		"EnvironmentFile=-/etc/vps-agent/broker.env", "ExecStart=/usr/local/bin/vps-agent-broker",
		"Restart=on-failure", "RestartSec=2", "RuntimeDirectory=vps-agent", "RuntimeDirectoryMode=0750",
		"StateDirectory=vps-agent", "StateDirectoryMode=0750", "UMask=0077", "PrivateTmp=yes",
		"ProtectHome=yes", "ProtectSystem=strict", "ReadWritePaths=/run/vps-agent /var/lib/vps-agent",
		"RestrictSUIDSGID=yes", "LockPersonality=yes", "", "[Install]", "WantedBy=multi-user.target", "",
	}, "\n")
	if err := writeRootFile("/etc/systemd/system/vps-agent-gateway.service", []byte(gatewayUnit), 0644); err != nil { return err }
	if err := writeRootFile("/etc/systemd/system/vps-agent-broker.service", []byte(brokerUnit), 0644); err != nil { return err }
	if err := os.MkdirAll("/etc/systemd/system/vps-agent-broker.service.d", 0755); err != nil { return err }
	paths := []string{"/run/vps-agent", "/var/lib/vps-agent"}
	paths = append(paths, writeRoots...)
	if packages {
		paths = append(paths, "/")
	} else {
		if identities { paths = append(paths, "/etc", "/home") }
		if firewall { paths = append(paths, "/etc/ufw") }
	}
	seen := map[string]bool{}
	var b strings.Builder
	b.WriteString("[Service]\n")
	for _, p := range paths {
		p = filepath.Clean(p)
		if seen[p] { continue }
		seen[p] = true
		b.WriteString("ReadWritePaths="+p+"\n")
	}
	return writeRootFile("/etc/systemd/system/vps-agent-broker.service.d/10-policy-paths.conf", []byte(b.String()), 0644)
}

func systemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil { return fmt.Errorf("systemctl %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err) }
	return nil
}

func printChatGPTTutorial(st installState) {
	fmt.Println("\n=== ChatGPT Web connection tutorial ===")
	fmt.Println("1. Open ChatGPT on the web.")
	fmt.Println("2. Open the Plugins/Apps developer connection flow available to your plan/workspace.")
	fmt.Println("3. Add the MCP endpoint:", st.Endpoint)
	fmt.Println("4. Configure the authentication method:", st.AuthMode)
	fmt.Println("5. Enable/select the app/plugin in a fresh chat.")
	fmt.Println("6. Ask ChatGPT: Call system.info on my VPS MCP and tell me the hostname.")
	fmt.Println("")
	fmt.Println("Current release note: full private MCP write/modify is documented by OpenAI for Business, Enterprise and Edu.")
	fmt.Println("Follow current official OpenAI UI/docs if labels differ from this release.")
}

func readInstallState() (installState, error) {
	var st installState
	raw, err := os.ReadFile("/var/lib/vps-agent/install-status.json")
	if err != nil { return st, err }
	err = json.Unmarshal(raw, &st)
	return st, err
}

func readAdminToken() (string, error) {
	raw, err := os.ReadFile("/etc/vps-agent/broker.env")
	if err != nil { return "", err }
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "VPS_AGENT_ADMIN_TOKEN=") {
			return strings.TrimPrefix(line, "VPS_AGENT_ADMIN_TOKEN="), nil
		}
	}
	return "", errors.New("admin token not found")
}

func verifyChatGPT(args []string) error {
	if err := requireRoot(); err != nil { return err }
	fs := flag.NewFlagSet("verify-chatgpt", flag.ContinueOnError)
	timeout := fs.Duration("timeout", 5*time.Minute, "maximum wait for a new ChatGPT MCP call")
	if err := fs.Parse(args); err != nil { return err }
	st, err := readInstallState()
	if err != nil { return err }
	if st.State == "complete" { fmt.Println("Installation already complete."); return nil }
	token, err := readAdminToken()
	if err != nil { return err }
	deadline := time.Now().Add(*timeout)
	fmt.Printf("Waiting for system.info from subject %q after audit seq %d...\n", st.Subject, st.BaselineAudit)
	for time.Now().Before(deadline) {
		payload, _ := json.Marshal(map[string]any{"after_seq": st.BaselineAudit, "limit": 500})
		resp, err := brokerCall(token, wire.Request{ID: "setup-audit-tail", Tool: "admin.audit.tail", Args: payload})
		if err == nil {
			var body map[string]any
			if json.Unmarshal(resp.Result, &body) == nil {
				if records, ok := body["events"].([]any); ok {
					for _, raw := range records {
						rec, _ := raw.(map[string]any)
						event, _ := rec["event"].(map[string]any)
						if event["subject"] == st.Subject && event["tool"] == "system.info" && event["decision"] == "allow" {
							seqFloat, _ := rec["seq"].(float64)
							st.State = "complete"
							st.VerifiedAt = time.Now().UTC()
							st.VerifiedSeq = int64(seqFloat)
							if err := writeJSON("/var/lib/vps-agent/install-status.json", st, 0600); err != nil { return err }
							fmt.Printf("ChatGPT/MCP connection verified at audit seq %d.\n", st.VerifiedSeq)
							fmt.Println("INSTALLATION COMPLETE")
							return nil
						}
					}
				}
			}
		}
		time.Sleep(time.Second)
	}
	return errors.New("verification timeout; installation remains incomplete")
}

func writeJSON(path string, value any, mode os.FileMode) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil { return err }
	raw = append(raw, '\n')
	return writeRootFile(path, raw, mode)
}

func showStatus() error {
	st, err := readInstallState()
	if err != nil { return err }
	raw, _ := json.MarshalIndent(st, "", "  ")
	fmt.Println(string(raw))
	return nil
}

func upgrade(args []string) error {
	if err := requireRoot(); err != nil { return err }
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	sourceDir := fs.String("source-dir", ".", "directory containing new binaries")
	if err := fs.Parse(args); err != nil { return err }
	if _, err := os.Stat("/etc/vps-agent/policy.yaml"); err != nil { return errors.New("no existing installation found") }
	if err := installBinaries(*sourceDir); err != nil { return err }
	if err := systemctl("daemon-reload"); err != nil { return err }
	if err := systemctl("restart", "vps-agent-broker.service", "vps-agent-gateway.service"); err != nil { return err }
	fmt.Println("Upgrade complete; existing policy and verification state preserved.")
	return nil
}

func uninstall(args []string) error {
	if err := requireRoot(); err != nil { return err }
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	purge := fs.Bool("purge", false, "also remove policy, secrets and state")
	yes := fs.Bool("yes", false, "confirm uninstall")
	if err := fs.Parse(args); err != nil { return err }
	if !*yes { return errors.New("uninstall requires --yes") }
	_ = systemctl("disable", "--now", "vps-agent-gateway.service", "vps-agent-broker.service")
	for _, p := range []string{
		"/etc/systemd/system/vps-agent-gateway.service",
		"/etc/systemd/system/vps-agent-broker.service",
		"/etc/systemd/system/vps-agent-broker.service.d/10-policy-paths.conf",
	} { _ = os.Remove(p) }
	_ = os.Remove("/etc/systemd/system/vps-agent-broker.service.d")
	for _, name := range []string{"vps-agent-gateway", "vps-agent-broker", "vps-agent", "vps-agent-setup"} {
		_ = os.Remove(filepath.Join("/usr/local/bin", name))
	}
	_ = systemctl("daemon-reload")
	if *purge {
		_ = os.RemoveAll("/etc/vps-agent")
		_ = os.RemoveAll("/var/lib/vps-agent")
	}
	fmt.Println("Uninstall complete.")
	return nil
}

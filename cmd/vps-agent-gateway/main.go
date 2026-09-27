package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/gateway"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	listen := getenv("VPS_AGENT_LISTEN", "127.0.0.1:8080")
	socket := os.Getenv("VPS_AGENT_BROKER_SOCKET")
	pocRoot := getenv("VPS_AGENT_POC_ROOT", "/tmp/vps-agent-poc")

	var exec gateway.Executor
	if socket == "" {
		if err := os.MkdirAll(pocRoot, 0o700); err != nil {
			slog.Error("fatal", "error", err); os.Exit(1)
		}
		fs, err := securefs.New([]string{pocRoot}, []string{pocRoot}, securefs.DefaultMaxBytes)
		if err != nil {
			slog.Error("fatal", "error", err); os.Exit(1)
		}
		exec = gateway.LocalExecutor{FS: fs}
		slog.Info("gateway_local_mode", "safe_root", pocRoot)
	} else {
		exec = gateway.BrokerExecutor{Client: ipc.Client{Socket: socket, Timeout: 15 * time.Second}}
		slog.Info("gateway_broker_configured", "socket", socket)
	}

	issuer := strings.TrimRight(os.Getenv("VPS_AGENT_OIDC_ISSUER"), "/")
	resource := os.Getenv("VPS_AGENT_OAUTH_RESOURCE")
	if resource == "" {
		resource = os.Getenv("VPS_AGENT_PUBLIC_URL")
	}
	metadataURL := os.Getenv("VPS_AGENT_RESOURCE_METADATA_URL")
	if metadataURL == "" && resource != "" {
		metadataURL = metadataURLForResource(resource)
	}

	authCfg := gateway.AuthConfig{
		Mode:                 getenv("VPS_AGENT_AUTH_MODE", "none"),
		StaticToken:          os.Getenv("VPS_AGENT_STATIC_TOKEN"),
		StaticSubject:        getenv("VPS_AGENT_STATIC_SUBJECT", "local-dev"),
		RequiredScopes:       splitScopes(os.Getenv("VPS_AGENT_REQUIRED_SCOPES")),
		ResourceMetadataURL:  metadataURL,
		ResourceIdentifier:   resource,
		InstanceID:           os.Getenv("VPS_AGENT_INSTANCE_ID"),
		InstanceName:         getenv("VPS_AGENT_INSTANCE_NAME", "vps-agent"),
		MaxConcurrentRequests: getenvInt("VPS_AGENT_MAX_CONCURRENT_REQUESTS", 64),
	}
	if issuer != "" {
		authCfg.AuthorizationServers = []string{issuer}
	}
	if (authCfg.Mode == "" || authCfg.Mode == "none") && !listenIsLoopback(listen) {
		slog.Error("unauthenticated_remote_bind_rejected", "listen", listen, "reason", "auth mode none is allowed only on loopback")
		os.Exit(1)
	}
	if os.Getenv("VPS_AGENT_PUBLIC_URL") != "" && authCfg.Mode != "integrated" {
		slog.Error("public_auth_rejected", "reason", "public MCP configuration requires integrated auth"); os.Exit(1)
	}
	if authCfg.Mode == "integrated" {
		if resource == "" || issuer == "" {
			slog.Error("integrated_auth_config_invalid", "reason", "OAuth resource/public URL and issuer required"); os.Exit(1)
		}
		verifier, err := gateway.NewIntegratedOIDCVerifier(context.Background(), issuer)
		if err != nil {
			slog.Error("integrated_auth_config_failed", "error", err); os.Exit(1)
		}
		authCfg.Verifier = verifier
	}

	handler := gateway.Handler(exec, authCfg)
	srv := &http.Server{
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // Streamable HTTP may legitimately outlive a normal request timeout.
		IdleTimeout:       2 * time.Minute,
	}
	slog.Info("gateway_start", "listen", listen, "auth_mode", authCfg.Mode, "instance_id", authCfg.InstanceID, "instance_name", authCfg.InstanceName)
	if err := srv.ListenAndServe(); err != nil { slog.Error("gateway_exit", "error", err); os.Exit(1) }
}


func metadataURLForResource(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/.well-known/oauth-protected-resource"
}

func listenIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func splitScopes(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func getenvInt(name string, fallback int) int {
	v := os.Getenv(name)
	if v == "" { return fallback }
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 { return fallback }
	return n
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

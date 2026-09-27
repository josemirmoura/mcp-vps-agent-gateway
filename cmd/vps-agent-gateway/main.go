package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/gateway"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
)

func main() {
	listen := getenv("VPS_AGENT_LISTEN", ":8080")
	socket := os.Getenv("VPS_AGENT_BROKER_SOCKET")
	pocRoot := getenv("VPS_AGENT_POC_ROOT", "/tmp/vps-agent-poc")

	var exec gateway.Executor
	if socket == "" {
		if err := os.MkdirAll(pocRoot, 0o700); err != nil {
			log.Fatal(err)
		}
		fs, err := securefs.New([]string{pocRoot}, []string{pocRoot}, securefs.DefaultMaxBytes)
		if err != nil {
			log.Fatal(err)
		}
		exec = gateway.LocalExecutor{FS: fs}
		log.Printf("starting in Gate 0 local mode; safe root=%s", pocRoot)
	} else {
		exec = gateway.BrokerExecutor{Client: ipc.Client{Socket: socket, Timeout: 15 * time.Second}}
		log.Printf("using broker socket %s", socket)
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
	}
	if issuer != "" {
		authCfg.AuthorizationServers = []string{issuer}
	}
	if authCfg.Mode == "oidc" {
		if resource == "" {
			log.Fatal("VPS_AGENT_OAUTH_RESOURCE or VPS_AGENT_PUBLIC_URL is required in oidc mode")
		}
		audience := os.Getenv("VPS_AGENT_OIDC_AUDIENCE")
		if audience == "" {
			audience = resource
		}
		verifier, err := gateway.NewOIDCVerifier(
			context.Background(),
			issuer,
			audience,
		)
		if err != nil {
			log.Fatalf("configure OIDC: %v", err)
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
	log.Printf("MCP gateway listening on %s/mcp auth=%s", listen, authCfg.Mode)
	log.Fatal(srv.ListenAndServe())
}

func metadataURLForResource(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/.well-known/oauth-protected-resource"
}

func splitScopes(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

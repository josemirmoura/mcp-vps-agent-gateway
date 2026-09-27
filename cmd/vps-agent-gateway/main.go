package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

	handler := gateway.Handler(exec, gateway.AuthConfig{
		Mode:          getenv("VPS_AGENT_AUTH_MODE", "none"),
		StaticToken:   os.Getenv("VPS_AGENT_STATIC_TOKEN"),
		StaticSubject: getenv("VPS_AGENT_STATIC_SUBJECT", "local-dev"),
	})

	srv := &http.Server{
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("MCP gateway listening on %s/mcp", listen)
	log.Fatal(srv.ListenAndServe())
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

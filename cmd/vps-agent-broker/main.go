package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/jobs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
)

func main() {
	policyFile := getenv("VPS_AGENT_POLICY", "/etc/vps-agent/policy.yaml")
	socket := getenv("VPS_AGENT_BROKER_SOCKET", "/run/vps-agent/broker.sock")
	dbFile := getenv("VPS_AGENT_STATE_DB", "/var/lib/vps-agent/state.db")

	cfg, err := policy.Load(policyFile)
	if err != nil {
		log.Fatal(err)
	}
	fs, err := securefs.New(cfg.FileRoots(false), cfg.FileRoots(true), securefs.DefaultMaxBytes)
	if err != nil {
		log.Fatal(err)
	}
	store, err := state.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	b := &broker.Broker{
		Policy: cfg, FS: fs, State: store,
		Services: broker.SystemdManager{}, Docker: broker.DockerCLI{},
		Jobs: &jobs.Manager{State: store, Runner: jobs.SystemdRunner{}},
		AdminToken: os.Getenv("VPS_AGENT_ADMIN_TOKEN"),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log.Printf("broker listening on unix://%s", socket)
	if err := ipc.NewServer(socket, b).Serve(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

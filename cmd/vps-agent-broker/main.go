package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/jobs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	policyFile := getenv("VPS_AGENT_POLICY", "/etc/vps-agent/policy.yaml")
	socket := getenv("VPS_AGENT_BROKER_SOCKET", "/run/vps-agent/broker.sock")
	dbFile := getenv("VPS_AGENT_STATE_DB", "/var/lib/vps-agent/state.db")

	cfg, err := policy.Load(policyFile)
	if err != nil {
		slog.Error("fatal", "error", err); os.Exit(1)
	}
	if physicalRoot := os.Getenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT"); physicalRoot != "" {
		wholeHost := os.Getenv("VPS_AGENT_WHOLE_HOST") == "1"
		if err := cfg.ValidatePhysicalScope(physicalRoot, wholeHost); err != nil {
			slog.Error("physical_scope_invalid", "root", physicalRoot, "whole_host", wholeHost, "error", err)
			os.Exit(1)
		}
	}
	fs, err := securefs.NewWithHostRoot(
		cfg.FileRoots(false),
		cfg.FileRoots(true),
		securefs.DefaultMaxBytes,
		os.Getenv("VPS_AGENT_HOST_ROOT"),
	)
	if err != nil {
		slog.Error("fatal", "error", err); os.Exit(1)
	}
	store, err := state.Open(dbFile)
	if err != nil {
		slog.Error("fatal", "error", err); os.Exit(1)
	}
	defer store.Close()

	b := &broker.Broker{
		Policy: cfg, FS: fs, State: store,
		Services: broker.SystemdManager{}, Docker: broker.DockerCLI{},
		Jobs:       &jobs.Manager{State: store, Runner: jobs.SystemdRunner{}},
		AdminToken: os.Getenv("VPS_AGENT_ADMIN_TOKEN"),
		ExpectedSubject: os.Getenv("VPS_AGENT_EXPECTED_SUBJECT"),
		InstanceID: os.Getenv("VPS_AGENT_INSTANCE_ID"),
		InstanceName: getenv("VPS_AGENT_INSTANCE_NAME", "vps-agent"),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	slog.Info("broker_start", "socket", socket, "instance_id", b.InstanceID, "instance_name", b.InstanceName, "policy", policyFile)
	// Isolate web approvals on a distinct Unix socket and dedicated volume.
	operatorSocket := os.Getenv("VPS_AGENT_OPERATOR_APPROVAL_SOCKET")
	if operatorSocket == "" && operatorToken() != "" {
		operatorSocket = "/run/portico-operator/operator.sock"
	}
	if operatorSocket != "" {
		if len(operatorToken()) < 32 {
			slog.Error("operator_approval_token_too_short")
			os.Exit(1)
		}
		operatorServer := ipc.NewServer(operatorSocket, &operatorBridge{broker: b, token: operatorToken()})
		operatorServer.SocketGroupGID = 65532
		operatorServer.AllowPeerUIDs(0, 65532)
		go func() {
			if err := operatorServer.Serve(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operator_approval_socket_failed", "error", err)
				cancel()
			}
		}()
	}
	server := ipc.NewServer(socket, b)
	server.AllowPeerUIDs(0, 65532)
	if gatewayUser, err := user.Lookup("vps-agent"); err == nil {
		if uid, err := strconv.ParseUint(gatewayUser.Uid, 10, 32); err == nil {
			server.AllowPeerUIDs(uint32(uid))
		}
	}
	if err := server.Serve(ctx); err != nil && ctx.Err() == nil {
		slog.Error("fatal", "error", err); os.Exit(1)
	}
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

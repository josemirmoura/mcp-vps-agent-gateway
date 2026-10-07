package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/cloudnode"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cloudURL := strings.TrimSpace(os.Getenv("PORTICO_CLOUD_URL"))
	if cloudURL == "" {
		fatal("PORTICO_CLOUD_URL is required", nil)
	}

	client, err := cloudnode.NewClient(cloudURL, nil)
	if err != nil {
		fatal("invalid Portico Cloud configuration", err)
	}

	statePath := getenv("PORTICO_CLOUD_STATE", "/var/lib/portico-cloud-node/state.json")
	nodeName := strings.TrimSpace(os.Getenv("PORTICO_CLOUD_NODE_NAME"))
	if nodeName == "" {
		nodeName, _ = os.Hostname()
	}
	if nodeName == "" {
		nodeName = "portico-node"
	}

	enrollmentToken, err := readEnrollmentToken()
	if err != nil {
		fatal("failed to read enrollment token", err)
	}

	identity, err := cloudnode.EnsureEnrollment(ctx, client, statePath, enrollmentToken, nodeName)
	if err != nil {
		fatal("Portico Cloud enrollment failed", err)
	}
	_ = os.Unsetenv("PORTICO_CLOUD_ENROLLMENT_TOKEN")
	enrollmentToken = ""

	brokerSubject := strings.TrimSpace(os.Getenv("PORTICO_CLOUD_BROKER_SUBJECT"))
	if brokerSubject == "" {
		brokerSubject = getenv("VPS_AGENT_SUBJECT", "operator")
	}
	brokerSocket := getenv("PORTICO_CLOUD_BROKER_SOCKET", "/run/vps-agent/broker.sock")

	heartbeatInterval := durationEnv("PORTICO_CLOUD_HEARTBEAT_INTERVAL", 30*time.Second)
	pollInterval := durationEnv("PORTICO_CLOUD_POLL_INTERVAL", 2*time.Second)
	renewInterval := durationEnv("PORTICO_CLOUD_RENEW_INTERVAL", 10*time.Second)
	brokerTimeout := durationEnv("PORTICO_CLOUD_BROKER_TIMEOUT", 15*time.Minute)

	slog.Info("portico_cloud_node_start",
		"node_id", identity.NodeID,
		"workspace_id", identity.WorkspaceID,
		"node_name", nodeName,
		"broker_subject", brokerSubject,
	)

	runner := cloudnode.Runner{
		Client:            client,
		Identity:          identity,
		Broker:            ipc.Client{Socket: brokerSocket, Timeout: brokerTimeout},
		BrokerSubject:     brokerSubject,
		HeartbeatInterval: heartbeatInterval,
		PollInterval:      pollInterval,
		RenewInterval:     renewInterval,
		BrokerTimeout:     brokerTimeout,
	}
	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fatal("Portico Cloud node connector stopped", err)
	}

	slog.Info("portico_cloud_node_stop", "node_id", identity.NodeID)
}

func readEnrollmentToken() (string, error) {
	if path := strings.TrimSpace(os.Getenv("PORTICO_CLOUD_ENROLLMENT_TOKEN_FILE")); path != "" {
		data, err := os.ReadFile(path) // #nosec G304 -- operator-selected secret file is an explicit connector input.
		if err != nil {
			return "", err
		}
		token := strings.TrimSpace(string(data))
		if token == "" {
			return "", errors.New("enrollment token file is empty")
		}
		return token, nil
	}
	return strings.TrimSpace(os.Getenv("PORTICO_CLOUD_ENROLLMENT_TOKEN")), nil
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		slog.Warn("invalid_duration_env", "name", name, "value", value, "fallback", fallback)
		return fallback
	}
	return duration
}

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func fatal(message string, err error) {
	if err != nil {
		slog.Error(message, "error", err)
	} else {
		slog.Error(message)
	}
	os.Exit(1)
}

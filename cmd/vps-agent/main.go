package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "grant":
		grant(os.Args[2:])
	case "revoke-all":
		revokeAll(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func common(fs *flag.FlagSet) (*string, *string) {
	socket := fs.String("socket", getenv("VPS_AGENT_BROKER_SOCKET", "/run/vps-agent/broker.sock"), "broker unix socket")
	token := fs.String("admin-token", os.Getenv("VPS_AGENT_ADMIN_TOKEN"), "operator token; prefer env VPS_AGENT_ADMIN_TOKEN")
	return socket, token
}

func grant(args []string) {
	fs := flag.NewFlagSet("grant", flag.ExitOnError)
	socket, token := common(fs)
	subject := fs.String("subject", "", "grant subject")
	caps := fs.String("cap", "", "comma-separated capabilities")
	ttl := fs.Duration("ttl", 30*time.Minute, "grant TTL")
	_ = fs.Parse(args)
	if *subject == "" || *caps == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "subject, cap and admin token are required")
		os.Exit(2)
	}
	payload, _ := json.Marshal(map[string]any{
		"subject": *subject,
		"capabilities": strings.Split(*caps, ","),
		"ttl_seconds": int64(ttl.Seconds()),
	})
	call(*socket, wire.Request{
		ID: "operator-grant", Tool: "admin.grant.issue", AdminToken: *token, Args: payload,
	})
}

func revokeAll(args []string) {
	fs := flag.NewFlagSet("revoke-all", flag.ExitOnError)
	socket, token := common(fs)
	_ = fs.Parse(args)
	if *token == "" {
		fmt.Fprintln(os.Stderr, "admin token is required")
		os.Exit(2)
	}
	call(*socket, wire.Request{ID: "operator-revoke-all", Tool: "admin.revoke_all", AdminToken: *token})
}

func call(socket string, req wire.Request) {
	resp, err := (ipc.Client{Socket: socket, Timeout: 10 * time.Second}).Call(context.Background(), req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(raw))
	if !resp.OK {
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: vps-agent <grant|revoke-all> [flags]")
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

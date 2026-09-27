package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	case "approvals":
		listApprovals(os.Args[2:])
	case "approve":
		decideApproval(os.Args[2:], true)
	case "deny":
		decideApproval(os.Args[2:], false)
	case "revoke-all":
		revokeAll(os.Args[2:])
	case "audit-status":
		auditStatus(os.Args[2:])
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

func listApprovals(args []string) {
	fs := flag.NewFlagSet("approvals", flag.ExitOnError)
	socket, token := common(fs)
	_ = fs.Parse(args)
	requireToken(*token)
	call(*socket, wire.Request{ID: "operator-approval-list", Tool: "admin.approval.list", AdminToken: *token})
}

func decideApproval(args []string, approve bool) {
	name := "approve"
	tool := "admin.approval.approve"
	if !approve {
		name = "deny"
		tool = "admin.approval.deny"
	}
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	socket, token := common(fs)
	requestID := fs.String("request", "", "approval request id")
	_ = fs.Parse(args)
	requireToken(*token)
	if *requestID == "" {
		fmt.Fprintln(os.Stderr, "request id is required")
		os.Exit(2)
	}
	payload, _ := json.Marshal(map[string]any{"request_id": *requestID})
	call(*socket, wire.Request{
		ID: "operator-" + name, Tool: tool, AdminToken: *token, Args: payload,
	})
}

func auditStatus(args []string) {
	fs := flag.NewFlagSet("audit-status", flag.ExitOnError)
	socket, token := common(fs)
	_ = fs.Parse(args)
	if *token == "" {
		fmt.Fprintln(os.Stderr, "admin token is required")
		os.Exit(2)
	}
	call(*socket, wire.Request{ID: "operator-audit-status", Tool: "admin.audit.status", AdminToken: *token})
}

func revokeAll(args []string) {
	fs := flag.NewFlagSet("revoke-all", flag.ExitOnError)
	socket, token := common(fs)
	_ = fs.Parse(args)
	requireToken(*token)
	call(*socket, wire.Request{ID: "operator-revoke-all", Tool: "admin.revoke_all", AdminToken: *token})
}

func requireToken(token string) {
	if token == "" {
		fmt.Fprintln(os.Stderr, "admin token is required")
		os.Exit(2)
	}
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
	fmt.Fprintln(os.Stderr, "usage: vps-agent <approvals|approve|deny|revoke-all> [flags]")
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

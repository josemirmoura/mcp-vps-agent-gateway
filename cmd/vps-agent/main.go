package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "health":
		health(os.Args[2:])
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
	case "audit-tail":
		auditTail(os.Args[2:])
	case "wait-tool":
		waitTool(os.Args[2:])
	case "state-check":
		stateCheck(os.Args[2:])
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


func callResponse(socket string, req wire.Request) (wire.Response, error) {
	resp, err := (ipc.Client{Socket: socket, Timeout: 10 * time.Second}).Call(context.Background(), req)
	if err != nil {
		return resp, err
	}
	if !resp.OK {
		if resp.Error != nil {
			return resp, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, fmt.Errorf("broker request failed")
	}
	return resp, nil
}

func waitTool(args []string) {
	fs := flag.NewFlagSet("wait-tool", flag.ExitOnError)
	socket, token := common(fs)
	subject := fs.String("subject", "", "expected authenticated subject")
	tool := fs.String("tool", "system.info", "expected MCP tool name")
	timeout := fs.Duration("timeout", 5*time.Minute, "maximum wait time")
	poll := fs.Duration("poll", time.Second, "poll interval")
	afterSeq := fs.Int64("after-seq", -1, "only accept matching audit events after this sequence; default uses current audit head")
	quiet := fs.Bool("quiet", false, "suppress human progress/result text; use exit status only")
	baselineOnly := fs.Bool("baseline-only", false, "print the validated current audit sequence and exit")
	_ = fs.Parse(args)
	requireToken(*token)
	if *subject == "" {
		fmt.Fprintln(os.Stderr, "subject is required")
		os.Exit(2)
	}

	statusResp, err := callResponse(*socket, wire.Request{
		ID: "operator-wait-tool-status", Tool: "admin.audit.status", AdminToken: *token,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var status state.AuditStatus
	if err := json.Unmarshal(statusResp.Result, &status); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !status.Valid {
		fmt.Fprintln(os.Stderr, "audit chain is invalid")
		os.Exit(1)
	}
	if *baselineOnly {
		fmt.Println(status.Events)
		return
	}

	baseline := status.Events
	if *afterSeq >= 0 {
		if *afterSeq > status.Events {
			fmt.Fprintln(os.Stderr, "after-seq is beyond the current audit head")
			os.Exit(2)
		}
		baseline = *afterSeq
	}
	if !*quiet {
		fmt.Printf("Waiting for MCP tool %q from subject %q after audit seq %d...\n", *tool, *subject, baseline)
	}

	deadline := time.Now().Add(*timeout)
	for time.Now().Before(deadline) {
		payload, _ := json.Marshal(map[string]any{"after_seq": baseline, "limit": 500})
		resp, err := callResponse(*socket, wire.Request{
			ID: "operator-wait-tool-tail", Tool: "admin.audit.tail", AdminToken: *token, Args: payload,
		})
		if err == nil {
			var body struct {
				Events []state.AuditRecord `json:"events"`
			}
			if json.Unmarshal(resp.Result, &body) == nil {
				for _, rec := range body.Events {
					if rec.Event.Subject == *subject && rec.Event.Tool == *tool && rec.Event.Decision == "allow" {
						if !*quiet {
							raw, _ := json.MarshalIndent(rec, "", "  ")
							fmt.Println(string(raw))
							fmt.Println("CHATGPT/MCP CONNECTION VERIFIED")
						}
						return
					}
				}
			}
		}
		time.Sleep(*poll)
	}
	if !*quiet {
		fmt.Fprintln(os.Stderr, "verification timeout; no matching audited tool call arrived")
	}
	os.Exit(1)
}

func stateCheck(args []string) {
	fs := flag.NewFlagSet("state-check", flag.ExitOnError)
	dbFile := fs.String("db", "", "path to a copied Broker SQLite state database")
	_ = fs.Parse(args)
	if *dbFile == "" {
		fmt.Fprintln(os.Stderr, "db is required")
		os.Exit(2)
	}
	store, err := state.Open(*dbFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()
	version, err := store.SchemaVersion(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	audit, err := store.AuditStatus(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, _ := json.MarshalIndent(map[string]any{
		"schema_version": version,
		"audit": audit,
	}, "", "  ")
	fmt.Println(string(raw))
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

func health(args []string) {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	socket, token := common(fs)
	_ = fs.Parse(args)
	requireToken(*token)
	call(*socket, wire.Request{ID: "operator-health", Tool: "admin.health", AdminToken: *token})
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


func auditTail(args []string) {
	fs := flag.NewFlagSet("audit-tail", flag.ExitOnError)
	socket, token := common(fs)
	after := fs.Int64("after", 0, "return audit records with sequence greater than this value")
	limit := fs.Int("limit", 100, "maximum records")
	_ = fs.Parse(args)
	requireToken(*token)
	payload, _ := json.Marshal(map[string]any{"after_seq": *after, "limit": *limit})
	call(*socket, wire.Request{ID: "operator-audit-tail", Tool: "admin.audit.tail", AdminToken: *token, Args: payload})
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
	fmt.Fprintln(os.Stderr, "usage: vps-agent <health|approvals|approve|deny|revoke-all|audit-status|audit-tail|wait-tool|state-check> [flags]")
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

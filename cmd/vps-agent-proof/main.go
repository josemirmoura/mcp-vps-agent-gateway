package main

import (
	"net/http"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Request struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Service       string `json:"service,omitempty"`
	Directory     string `json:"directory,omitempty"`
	HTMLPath      string `json:"html_path,omitempty"`
	HTMLContent   string `json:"html_content,omitempty"`
	SecondPath    string `json:"second_path,omitempty"`
	SecondContent string `json:"second_content,omitempty"`
}

type Evidence struct {
	RequestID  string         `json:"request_id"`
	Subject    string         `json:"subject"`
	Type       string         `json:"type"`
	Endpoint   string         `json:"endpoint"`
	StartedAt  string         `json:"started_at"`
	FinishedAt string         `json:"finished_at"`
	Success    bool           `json:"success"`
	Steps      []EvidenceStep `json:"steps"`
}

type EvidenceStep struct {
	Tool              string `json:"tool"`
	ExpectedDenial    bool   `json:"expected_denial,omitempty"`
	IsError           bool   `json:"is_error"`
	StructuredContent any    `json:"structured_content,omitempty"`
}

func main() {
	var requestFile, endpoint, token, output string
	flag.StringVar(&requestFile, "request", "proof/request.json", "proof request JSON")
	flag.StringVar(&endpoint, "endpoint", "http://127.0.0.1:8080/mcp", "MCP endpoint")
	flag.StringVar(&token, "token", os.Getenv("VPS_AGENT_PROOF_TOKEN"), "Bearer token")
	flag.StringVar(&output, "output", "proof/evidence.json", "evidence output JSON")
	flag.Parse()

	// #nosec G304 -- requestFile is an explicit operator-supplied proof-harness input, not an MCP-controlled path.\n\traw, err := os.ReadFile(requestFile)
	must(err)
	var req Request
	must(json.Unmarshal(raw, &req))
	if req.ID == "" || req.Subject == "" || req.Type == "" {
		must(errors.New("request id, subject and type are required"))
	}

	started := time.Now().UTC()
	ev := Evidence{
		RequestID: req.ID, Subject: req.Subject, Type: req.Type,
		Endpoint: endpoint, StartedAt: started.Format(time.RFC3339Nano),
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "vps-agent-proof", Version: "v1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: endpoint}
	if token != "" {
		transport.HTTPClient = &http.Client{Transport: bearerRoundTripper{token: token}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, transport, nil)
	must(err)
	defer session.Close()

	switch req.Type {
	case "file_write_read":
		if req.Path == "" {
			must(errors.New("path is required"))
		}
		ev.Steps = append(ev.Steps, call(ctx, session, "file.write_test", map[string]any{
			"path": req.Path, "content": req.Content,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "file.read_test", map[string]any{
			"path": req.Path,
		}, false))
	case "file_read_denied":
		if req.Path == "" {
			must(errors.New("path is required"))
		}
		ev.Steps = append(ev.Steps, call(ctx, session, "file.read_test", map[string]any{
			"path": req.Path,
		}, true))
	case "directory_html_and_file":
		if req.Directory == "" || req.HTMLPath == "" || req.SecondPath == "" {
			must(errors.New("directory, html_path and second_path are required"))
		}
		ev.Steps = append(ev.Steps, call(ctx, session, "file.mkdir", map[string]any{
			"path": req.Directory,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "file.write", map[string]any{
			"path": req.HTMLPath, "content": req.HTMLContent,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "file.read", map[string]any{
			"path": req.HTMLPath,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "file.write", map[string]any{
			"path": req.SecondPath, "content": req.SecondContent,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "file.read", map[string]any{
			"path": req.SecondPath,
		}, false))
	case "service_restart":
		if req.Service == "" {
			must(errors.New("service is required"))
		}
		ev.Steps = append(ev.Steps, call(ctx, session, "service.status", map[string]any{
			"name": req.Service,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "service.restart", map[string]any{
			"name": req.Service,
		}, false))
		ev.Steps = append(ev.Steps, call(ctx, session, "service.status", map[string]any{
			"name": req.Service,
		}, false))
	default:
		must(fmt.Errorf("unsupported proof request type %q", req.Type))
	}

	ev.Success = true
	for _, step := range ev.Steps {
		if step.ExpectedDenial {
			if !step.IsError {
				ev.Success = false
			}
		} else if step.IsError {
			ev.Success = false
		}
	}
	ev.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)

	must(os.MkdirAll(dir(output), 0o750))
	encoded, err := json.MarshalIndent(ev, "", "  ")
	must(err)
	must(os.WriteFile(output, encoded, 0o600))
	fmt.Println(string(encoded))
	if !ev.Success {
		os.Exit(1)
	}
}

func call(ctx context.Context, session *mcp.ClientSession, tool string, args map[string]any, expectedDenial bool) EvidenceStep {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s protocol error: %v\n", tool, err)
		return EvidenceStep{Tool: tool, ExpectedDenial: expectedDenial, IsError: true, StructuredContent: map[string]any{"protocol_error": err.Error()}}
	}
	return EvidenceStep{
		Tool: tool, ExpectedDenial: expectedDenial, IsError: result.IsError,
		StructuredContent: result.StructuredContent,
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "."
}

// RoundTripper wrapper avoids coupling proof code to server internals.
type bearerRoundTripper struct {
	token string
}

func (h bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+h.token)
	return http.DefaultTransport.RoundTrip(req)
}

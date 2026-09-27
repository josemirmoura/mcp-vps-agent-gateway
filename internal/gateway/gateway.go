package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Executor interface {
	Call(context.Context, wire.Request) (wire.Response, error)
}

type BrokerExecutor struct {
	Client ipc.Client
}

func (e BrokerExecutor) Call(ctx context.Context, req wire.Request) (wire.Response, error) {
	return e.Client.Call(ctx, req)
}

type LocalExecutor struct {
	FS *securefs.Manager
}

func (e LocalExecutor) Call(_ context.Context, req wire.Request) (wire.Response, error) {
	switch req.Tool {
	case "system.info":
		host, _ := os.Hostname()
		return response(req.ID, map[string]any{
			"hostname": host, "goos": runtime.GOOS, "goarch": runtime.GOARCH,
			"cpus": runtime.NumCPU(), "mode": "gate0-local",
		}), nil
	case "file.read_test":
		data, err := e.FS.ReadFile(req.Resource)
		if err != nil {
			return wire.ErrorResponse(req.ID, "permission_denied", err.Error()), nil
		}
		return response(req.ID, map[string]any{"content": string(data), "bytes": len(data)}), nil
	case "file.write_test":
		var in struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(req.Args, &in); err != nil {
			return wire.ErrorResponse(req.ID, "invalid_args", err.Error()), nil
		}
		if err := e.FS.WriteFileAtomic(req.Resource, []byte(in.Content)); err != nil {
			return wire.ErrorResponse(req.ID, "permission_denied", err.Error()), nil
		}
		return response(req.ID, map[string]any{"written": true, "bytes": len(in.Content)}), nil
	case "system.health":
		return response(req.ID, map[string]any{"ok": true, "mode": "gate0-local"}), nil
	case "permissions.status":
		return response(req.ID, map[string]any{"mode": "gate0-local", "full_enabled": false}), nil
	default:
		return wire.ErrorResponse(req.ID, "broker_required", "tool requires privileged broker"), nil
	}
}

func response(id string, value any) wire.Response {
	raw, _ := json.Marshal(value)
	return wire.Response{ID: id, OK: true, Result: raw}
}

type subjectKey struct{}

func SubjectFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(subjectKey{}).(string); ok && v != "" {
		return v
	}
	return "anonymous"
}

type Server struct {
	Executor Executor
}

type systemInfoOutput struct {
	Hostname string `json:"hostname,omitempty"`
	GOOS     string `json:"goos,omitempty"`
	GOARCH   string `json:"goarch,omitempty"`
	CPUs     int    `json:"cpus,omitempty"`
	Mode     string `json:"mode,omitempty"`
}

type fileReadInput struct {
	Path string `json:"path" jsonschema:"absolute path inside the configured safe root"`
}

type fileReadOutput struct {
	Content string `json:"content"`
	Bytes   int    `json:"bytes"`
}

type fileWriteInput struct {
	Path    string `json:"path" jsonschema:"absolute path inside the configured safe root"`
	Content string `json:"content" jsonschema:"UTF-8 text content to write"`
}

type fileWriteOutput struct {
	Written bool `json:"written"`
	Bytes   int  `json:"bytes"`
}

type serviceInput struct {
	Name string `json:"name" jsonschema:"canonical systemd unit name"`
}

type serviceOutput struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Restarted bool   `json:"restarted,omitempty"`
}

type permissionsOutput struct {
	Mode        string `json:"mode"`
	FullEnabled bool   `json:"full_enabled"`
}

type dockerInput struct {
	Name string `json:"name" jsonschema:"canonical Docker container or resource name"`
}

type dockerLogsInput struct {
	Name  string `json:"name" jsonschema:"canonical Docker container or resource name"`
	Lines int    `json:"lines,omitempty" jsonschema:"number of log lines, maximum 1000"`
}

type dockerActionInput struct {
	Name        string `json:"name" jsonschema:"canonical Docker container or resource name"`
	Action      string `json:"action" jsonschema:"typed action; currently restart"`
	OperationID string `json:"operation_id,omitempty" jsonschema:"stable retry identity when the client can preserve one"`
}

type jobInput struct {
	JobID string `json:"job_id" jsonschema:"durable job identifier"`
}

type jobTailInput struct {
	JobID string `json:"job_id" jsonschema:"durable job identifier"`
	Lines int    `json:"lines,omitempty" jsonschema:"number of log lines, maximum 1000"`
}

type elevationInput struct {
	Capabilities []string `json:"capabilities" jsonschema:"explicit capabilities requested for temporary elevation"`
	TTLSeconds   int64    `json:"ttl_seconds" jsonschema:"requested grant lifetime in seconds"`
}

func NewMCPServer(exec Executor) *mcp.Server {
	s := &Server{Executor: exec}
	server := mcp.NewServer(&mcp.Implementation{Name: "mcp-vps-agent-gateway", Version: "v0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "system.info", Description: "Return non-sensitive host/runtime information."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, systemInfoOutput, error) {
			var out systemInfoOutput
			if err := s.call(ctx, "system.info", "", "", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.read_test", Description: "Read a text file only from the disposable Gate 0 safe root."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileReadInput) (*mcp.CallToolResult, fileReadOutput, error) {
			var out fileReadOutput
			if err := s.call(ctx, "file.read_test", in.Path, "", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.write_test", Description: "Write a text file only inside the disposable Gate 0 safe root."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileWriteInput) (*mcp.CallToolResult, fileWriteOutput, error) {
			var out fileWriteOutput
			args, _ := json.Marshal(map[string]any{"content": in.Content})
			if err := s.call(ctx, "file.write_test", in.Path, "", args, &out, true, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "service.status", Description: "Return status for a systemd service allowed by server-side policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serviceInput) (*mcp.CallToolResult, serviceOutput, error) {
			var out serviceOutput
			if err := s.call(ctx, "service.status", in.Name, "status", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "service.restart", Description: "Restart one systemd service only when server-side policy permits it."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serviceInput) (*mcp.CallToolResult, serviceOutput, error) {
			var out serviceOutput
			if err := s.call(ctx, "service.restart", in.Name, "restart", nil, &out, true, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "system.health", Description: "Return non-secret Gateway/Broker health information."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "system.health", "", "", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "docker.inspect", Description: "Inspect one Docker resource allowed by server-side policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in dockerInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "docker.inspect", in.Name, "inspect", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "docker.logs", Description: "Read bounded logs from one Docker resource allowed by server-side policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in dockerLogsInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"lines": in.Lines})
			if err := s.call(ctx, "docker.logs", in.Name, "logs", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "docker.action", Description: "Perform one typed Docker mutation when server-side policy permits it."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in dockerActionInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"action": in.Action})
			if err := s.call(ctx, "docker.action", in.Name, in.Action, args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "job.status", Description: "Return durable job state for the authenticated subject."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in jobInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "job.status", in.JobID, "status", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "job.tail", Description: "Read bounded output for a durable job owned by the authenticated subject."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in jobTailInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"lines": in.Lines})
			if err := s.call(ctx, "job.tail", in.JobID, "tail", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "job.cancel", Description: "Cancel a durable job owned by the authenticated subject."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in jobInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "job.cancel", in.JobID, "cancel", nil, &out, true, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "permissions.request_elevation", Description: "Create a pending temporary-elevation request. This tool cannot approve its own request."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in elevationInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(in)
			if err := s.call(ctx, "permissions.request_elevation", "", "request", args, &out, true, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "permissions.status", Description: "Report the active server-side permission mode without changing it."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, permissionsOutput, error) {
			var out permissionsOutput
			if err := s.call(ctx, "permissions.status", "", "", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})
	return server
}

func (s *Server) call(ctx context.Context, tool, resource, action string, args []byte, out any, write bool, operationID string) error {
	id, err := randomID()
	if err != nil {
		return err
	}
	req := wire.Request{
		ID: id, Subject: SubjectFromContext(ctx), Tool: tool, Resource: resource,
		Action: action, Args: args,
	}
	if write {
		req.InvocationID = operationID
		if req.InvocationID == "" {
			req.InvocationID = id
		}
	}
	resp, err := s.Executor.Call(ctx, req)
	if err != nil {
		return err
	}
	if !resp.OK {
		if resp.Error == nil {
			return errors.New("broker denied request")
		}
		return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
	}
	if out != nil && len(resp.Result) != 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("decode broker response: %w", err)
		}
	}
	return nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "inv_" + hex.EncodeToString(b[:]), nil
}

func Handler(exec Executor, auth AuthConfig) http.Handler {
	server := NewMCPServer(exec)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          1 << 20,
	})

	mux := http.NewServeMux()
	mux.Handle("/mcp", auth.Wrap(mcpHandler))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	return mux
}

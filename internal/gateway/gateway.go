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
	"time"

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
	Metrics  *runtimeMetrics
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

type fileMkdirInput struct {
	Path string `json:"path" jsonschema:"absolute directory path inside an authorized writable root"`
}

type fileMkdirOutput struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
}

type fileListInput struct {
	Path  string `json:"path" jsonschema:"absolute authorized directory path"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum entries, capped at 1000"`
}

type filePatchInput struct {
	Path           string `json:"path"`
	OldText        string `json:"old_text"`
	NewText        string `json:"new_text"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	OperationID    string `json:"operation_id,omitempty"`
}

type fileDestinationInput struct {
	Path        string `json:"path"`
	Destination string `json:"destination"`
	OperationID string `json:"operation_id,omitempty"`
}

type fileRemoveInput struct {
	Path        string `json:"path"`
	Recursive   bool   `json:"recursive,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
}

type fileChmodInput struct {
	Path        string `json:"path"`
	Mode        uint32 `json:"mode" jsonschema:"permission bits as integer, e.g. 511 for 0777"`
	OperationID string `json:"operation_id,omitempty"`
}

type fileChownInput struct {
	Path        string `json:"path"`
	UID         int    `json:"uid"`
	GID         int    `json:"gid"`
	OperationID string `json:"operation_id,omitempty"`
}

type serviceInput struct {
	Name string `json:"name" jsonschema:"canonical systemd unit name"`
}

type serviceOutput struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Action  string `json:"action,omitempty"`
}

type serviceLogsInput struct {
	Name  string `json:"name"`
	Lines int    `json:"lines,omitempty"`
}

type serviceActionInput struct {
	Name        string `json:"name"`
	OperationID string `json:"operation_id,omitempty"`
}

type limitInput struct {
	Limit int `json:"limit,omitempty"`
}

type processInspectInput struct {
	PID int `json:"pid"`
}

type networkCheckInput struct {
	Destination    string `json:"destination" jsonschema:"allowed host:port destination"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
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
	Action      string `json:"action" jsonschema:"typed action: start, stop, or restart"`
	OperationID string `json:"operation_id,omitempty" jsonschema:"stable retry identity when the client can preserve one"`
}

type composeInput struct {
	ProjectDir  string `json:"project_dir" jsonschema:"absolute Docker Compose project directory"`
	OperationID string `json:"operation_id,omitempty"`
}

type shellExecInput struct {
	Command        string `json:"command" jsonschema:"shell command to execute in the server-enforced sandbox"`
	CWD            string `json:"cwd" jsonschema:"absolute working directory inside authorized shell roots"`
	RuntimeSeconds int    `json:"runtime_seconds,omitempty"`
	MemoryBytes    int64  `json:"memory_bytes,omitempty"`
	TasksMax       int    `json:"tasks_max,omitempty"`
	OperationID    string `json:"operation_id,omitempty"`
}

type adminShellExecInput struct {
	Command        string `json:"command"`
	CWD            string `json:"cwd"`
	RuntimeSeconds int    `json:"runtime_seconds,omitempty"`
	MemoryBytes    int64  `json:"memory_bytes,omitempty"`
	TasksMax       int    `json:"tasks_max,omitempty"`
	OperationID    string `json:"operation_id,omitempty"`
	GrantID        string `json:"grant_id" jsonschema:"human-approved temporary grant containing shell.admin"`
}

type jobInput struct {
	JobID string `json:"job_id" jsonschema:"durable job identifier"`
}

type jobTailInput struct {
	JobID string `json:"job_id" jsonschema:"durable job identifier"`
	Lines int    `json:"lines,omitempty" jsonschema:"number of log lines, maximum 1000"`
}

type packageActionInput struct {
	Name        string `json:"name,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
}

type userActionInput struct {
	Name        string `json:"name"`
	CreateHome  bool   `json:"create_home,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
}

type groupActionInput struct {
	Name        string `json:"name"`
	OperationID string `json:"operation_id,omitempty"`
}

type firewallActionInput struct {
	Action      string `json:"action" jsonschema:"allow, deny, delete_allow, or delete_deny"`
	Port        string `json:"port"`
	Protocol    string `json:"protocol,omitempty" jsonschema:"tcp or udp"`
	Source      string `json:"source,omitempty" jsonschema:"optional source IP/CIDR"`
	OperationID string `json:"operation_id,omitempty"`
}

type elevationInput struct {
	Capabilities []string `json:"capabilities" jsonschema:"explicit capabilities requested for temporary elevation"`
	TTLSeconds   int64    `json:"ttl_seconds" jsonschema:"requested grant lifetime in seconds"`
}

func NewMCPServer(exec Executor) *mcp.Server {
	return newMCPServer(exec, nil)
}

func newMCPServer(exec Executor, metrics *runtimeMetrics) *mcp.Server {
	s := &Server{Executor: exec, Metrics: metrics}
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

	mcp.AddTool(server, &mcp.Tool{Name: "file.read", Description: "Read a bounded text file only from server-authorized filesystem roots."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileReadInput) (*mcp.CallToolResult, fileReadOutput, error) {
			var out fileReadOutput
			if err := s.call(ctx, "file.read", in.Path, "", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.write", Description: "Atomically write a bounded text file only inside server-authorized writable roots."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileWriteInput) (*mcp.CallToolResult, fileWriteOutput, error) {
			var out fileWriteOutput
			args, _ := json.Marshal(map[string]any{"content": in.Content})
			if err := s.call(ctx, "file.write", in.Path, "", args, &out, true, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.mkdir", Description: "Create an authorized directory tree without following symlink components."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileMkdirInput) (*mcp.CallToolResult, fileMkdirOutput, error) {
			var out fileMkdirOutput
			if err := s.call(ctx, "file.mkdir", in.Path, "mkdir", nil, &out, true, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.list", Description: "List entries from an authorized directory."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileListInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"limit": in.Limit})
			if err := s.call(ctx, "file.list", in.Path, "list", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.stat", Description: "Return metadata for an authorized filesystem path without following it outside policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileReadInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "file.stat", in.Path, "stat", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.hash", Description: "Compute SHA-256 for an authorized regular file."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileReadInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "file.hash", in.Path, "hash", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.patch", Description: "Patch exactly one text occurrence in an authorized file, optionally guarded by SHA-256."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in filePatchInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"old_text": in.OldText, "new_text": in.NewText, "expected_sha256": in.ExpectedSHA256})
			if err := s.call(ctx, "file.patch", in.Path, "patch", args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.copy", Description: "Copy an authorized regular file to an authorized writable destination."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileDestinationInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"destination": in.Destination})
			if err := s.call(ctx, "file.copy", in.Path, "copy", args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.move", Description: "Move or rename an authorized path within writable scope."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileDestinationInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"destination": in.Destination})
			if err := s.call(ctx, "file.move", in.Path, "move", args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.remove", Description: "Delete an authorized file or empty directory. Recursive deletion requires a separate policy capability."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileRemoveInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"recursive": in.Recursive})
			if err := s.call(ctx, "file.remove", in.Path, "remove", args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.chmod", Description: "Change permission bits of an authorized path when explicitly enabled by policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileChmodInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"mode": in.Mode})
			if err := s.call(ctx, "file.chmod", in.Path, "chmod", args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "file.chown", Description: "Change numeric ownership of an authorized path when explicitly enabled by policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fileChownInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"uid": in.UID, "gid": in.GID})
			if err := s.call(ctx, "file.chown", in.Path, "chown", args, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "service.list", Description: "List only systemd services visible through server-side policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "service.list", "", "list", nil, &out, false, ""); err != nil {
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

	mcp.AddTool(server, &mcp.Tool{Name: "service.logs", Description: "Read bounded journal logs for an allowed systemd service."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serviceLogsInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"lines": in.Lines})
			if err := s.call(ctx, "service.logs", in.Name, "logs", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	for _, action := range []string{"start", "stop", "restart", "reload", "enable", "disable"} {
		action := action
		mcp.AddTool(server, &mcp.Tool{Name: "service." + action, Description: "Perform the typed systemd " + action + " action when server-side policy permits it."},
			func(ctx context.Context, _ *mcp.CallToolRequest, in serviceActionInput) (*mcp.CallToolResult, serviceOutput, error) {
				var out serviceOutput
				if err := s.call(ctx, "service."+action, in.Name, action, nil, &out, true, in.OperationID); err != nil {
					return nil, out, err
				}
				return nil, out, nil
			})
	}

	mcp.AddTool(server, &mcp.Tool{Name: "system.health", Description: "Return non-secret Gateway/Broker health information."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "system.health", "", "", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "system.disk", Description: "Return bounded filesystem capacity diagnostics when policy permits."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "system.disk", "", "inspect", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "system.memory", Description: "Return bounded non-secret memory diagnostics when policy permits."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "system.memory", "", "inspect", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "process.list", Description: "List bounded process metadata without command-line arguments or environment."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in limitInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"limit": in.Limit})
			if err := s.call(ctx, "process.list", "", "inspect", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "process.inspect", Description: "Inspect a process using a safe subset of /proc status fields."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in processInspectInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "process.inspect", fmt.Sprint(in.PID), "inspect", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "network.listen", Description: "Return bounded listening socket diagnostics when explicitly enabled."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in limitInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"limit": in.Limit})
			if err := s.call(ctx, "network.listen", "", "inspect", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "network.check", Description: "Test TCP reachability only to a destination allowed by network policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in networkCheckInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"timeout_seconds": in.TimeoutSeconds})
			if err := s.call(ctx, "network.check", in.Destination, "check", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "docker.list", Description: "List Docker resources visible through server-side policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "docker.list", "", "list", nil, &out, false, ""); err != nil {
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

	mcp.AddTool(server, &mcp.Tool{Name: "compose.validate", Description: "Validate an authorized Docker Compose project without returning expanded configuration or secrets."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in composeInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "compose.validate", in.ProjectDir, "validate", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	for _, action := range []string{"pull", "up", "down"} {
		action := action
		mcp.AddTool(server, &mcp.Tool{Name: "compose." + action, Description: "Perform the typed Docker Compose " + action + " action when policy permits it."},
			func(ctx context.Context, _ *mcp.CallToolRequest, in composeInput) (*mcp.CallToolResult, map[string]any, error) {
				var out map[string]any
				if err := s.call(ctx, "compose."+action, in.ProjectDir, action, nil, &out, true, in.OperationID); err != nil {
					return nil, out, err
				}
				return nil, out, nil
			})
	}

	for _, toolName := range []string{"shell.exec", "job.start"} {
		toolName := toolName
		mcp.AddTool(server, &mcp.Tool{Name: toolName, Description: "Start a durable sandboxed command job inside server-authorized shell and filesystem scope."},
			func(ctx context.Context, _ *mcp.CallToolRequest, in shellExecInput) (*mcp.CallToolResult, map[string]any, error) {
				var out map[string]any
				args, _ := json.Marshal(map[string]any{
					"command": in.Command, "cwd": in.CWD, "runtime_seconds": in.RuntimeSeconds,
					"memory_bytes": in.MemoryBytes, "tasks_max": in.TasksMax,
				})
				if err := s.call(ctx, toolName, in.CWD, "start", args, &out, true, in.OperationID); err != nil {
					return nil, out, err
				}
				return nil, out, nil
			})
	}

	mcp.AddTool(server, &mcp.Tool{Name: "shell.exec_admin", Description: "Start a temporary human-approved administrative shell job. Full mode and a valid shell.admin grant are required."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in adminShellExecInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{
				"command": in.Command, "cwd": in.CWD, "runtime_seconds": in.RuntimeSeconds,
				"memory_bytes": in.MemoryBytes, "tasks_max": in.TasksMax,
			})
			if err := s.callWithGrant(ctx, "shell.exec_admin", in.CWD, "start", args, &out, in.OperationID, in.GrantID); err != nil {
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

	mcp.AddTool(server, &mcp.Tool{Name: "package.list", Description: "List only installed packages visible through server-side package policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in limitInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"limit": in.Limit})
			if err := s.call(ctx, "package.list", "", "list", args, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "package.update", Description: "Run the host package index update only when explicitly enabled by policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in packageActionInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "package.update", "", "update", nil, &out, true, in.OperationID); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	for _, action := range []string{"install", "remove"} {
		action := action
		mcp.AddTool(server, &mcp.Tool{Name: "package." + action, Description: "Perform typed APT " + action + " only for packages permitted by policy."},
			func(ctx context.Context, _ *mcp.CallToolRequest, in packageActionInput) (*mcp.CallToolResult, map[string]any, error) {
				var out map[string]any
				if err := s.call(ctx, "package."+action, in.Name, action, nil, &out, true, in.OperationID); err != nil {
					return nil, out, err
				}
				return nil, out, nil
			})
	}

	mcp.AddTool(server, &mcp.Tool{Name: "user.list", Description: "List only users visible through identity policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "user.list", "", "list", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "user.inspect", Description: "Inspect one policy-authorized local user."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in userActionInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "user.inspect", in.Name, "inspect", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	for _, action := range []string{"add", "delete", "lock", "unlock"} {
		action := action
		mcp.AddTool(server, &mcp.Tool{Name: "user." + action, Description: "Perform typed local-user " + action + " only when policy permits it."},
			func(ctx context.Context, _ *mcp.CallToolRequest, in userActionInput) (*mcp.CallToolResult, map[string]any, error) {
				var out map[string]any
				args, _ := json.Marshal(map[string]any{"create_home": in.CreateHome})
				if err := s.call(ctx, "user."+action, in.Name, action, args, &out, true, in.OperationID); err != nil {
					return nil, out, err
				}
				return nil, out, nil
			})
	}

	mcp.AddTool(server, &mcp.Tool{Name: "group.list", Description: "List only groups visible through identity policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "group.list", "", "list", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "group.inspect", Description: "Inspect one policy-authorized local group."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in groupActionInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "group.inspect", in.Name, "inspect", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	for _, action := range []string{"add", "delete"} {
		action := action
		mcp.AddTool(server, &mcp.Tool{Name: "group." + action, Description: "Perform typed local-group " + action + " only when policy permits it."},
			func(ctx context.Context, _ *mcp.CallToolRequest, in groupActionInput) (*mcp.CallToolResult, map[string]any, error) {
				var out map[string]any
				if err := s.call(ctx, "group."+action, in.Name, action, nil, &out, true, in.OperationID); err != nil {
					return nil, out, err
				}
				return nil, out, nil
			})
	}

	mcp.AddTool(server, &mcp.Tool{Name: "firewall.status", Description: "Return UFW status only when firewall inspection is enabled by policy."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			if err := s.call(ctx, "firewall.status", "", "status", nil, &out, false, ""); err != nil {
				return nil, out, err
			}
			return nil, out, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "firewall.action", Description: "Apply a structured UFW allow/deny/delete rule only when policy permits it."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in firewallActionInput) (*mcp.CallToolResult, map[string]any, error) {
			var out map[string]any
			args, _ := json.Marshal(map[string]any{"action": in.Action, "port": in.Port, "protocol": in.Protocol, "source": in.Source})
			if err := s.call(ctx, "firewall.action", in.Port, in.Action, args, &out, true, in.OperationID); err != nil {
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


func (s *Server) callWithGrant(ctx context.Context, tool, resource, action string, args []byte, out any, operationID, grantID string) (err error) {
	started := time.Now()
	if s.Metrics != nil { defer func() { s.Metrics.observeTool(tool, started, err) }() }
	id, err := randomID()
	if err != nil {
		return err
	}
	if operationID == "" {
		operationID = id
	}
	req := wire.Request{
		ID: id, Subject: SubjectFromContext(ctx), Tool: tool, Resource: resource,
		Action: action, Args: args, InvocationID: operationID, GrantID: grantID,
	}
	resp, err := s.Executor.Call(ctx, req)
	if err != nil {
		if s.Metrics != nil { s.Metrics.observeIPCFailure() }
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

func (s *Server) call(ctx context.Context, tool, resource, action string, args []byte, out any, write bool, operationID string) (err error) {
	started := time.Now()
	if s.Metrics != nil { defer func() { s.Metrics.observeTool(tool, started, err) }() }
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
		if s.Metrics != nil { s.Metrics.observeIPCFailure() }
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
	metrics := newRuntimeMetrics()
	server := newMCPServer(exec, metrics)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          1 << 20,
	})

	mux := http.NewServeMux()
	if metadataHandler := auth.ProtectedResourceMetadataHandler(); metadataHandler != nil {
		mux.Handle("/.well-known/oauth-protected-resource", metadataHandler)
	}
	mux.Handle("/mcp", metrics.wrapHTTP(auth.InstanceID, auth.InstanceName, limitConcurrent(auth.MaxConcurrentRequests, auth.Wrap(mcpHandler))))
	mux.Handle("/metrics", auth.Wrap(metrics.handler(auth.InstanceID, auth.InstanceName)))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	return mux
}

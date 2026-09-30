package gateway

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// addAnnotatedTool makes the MCP metadata contract fail closed: every
// model-visible tool must either provide explicit annotations at registration
// time or have a deliberate classification here.
//
// These annotations are client-facing UX/safety hints only. They never replace
// Broker authorization, policy checks, scoped grants or human approvals.
func addAnnotatedTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	if tool.Annotations == nil {
		annotations, ok := annotationsForTool(tool.Name)
		if !ok {
			panic(fmt.Sprintf("Portico MCP tool %q has no deliberate MCP annotation classification", tool.Name))
		}
		tool.Annotations = annotations
	}
	mcp.AddTool(server, tool, handler)
}

func annotations(readOnly, destructive, idempotent, openWorld bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    readOnly,
		DestructiveHint: boolPtr(destructive),
		IdempotentHint:  idempotent,
		OpenWorldHint:   boolPtr(openWorld),
	}
}

func annotationsForTool(name string) (*mcp.ToolAnnotations, bool) {
	// Pure inspection and local inventory. These do not mutate host state and
	// do not contact external entities.
	switch name {
	case "system.info", "system.health", "system.disk", "system.memory",
		"file.read_test", "file.read", "file.list", "file.stat", "file.hash",
		"service.list", "service.status", "service.logs",
		"process.list", "process.inspect", "network.listen",
		"docker.list", "docker.inspect", "docker.logs",
		"compose.validate",
		"job.status", "job.tail",
		"package.list",
		"user.list", "user.inspect",
		"group.list", "group.inspect",
		"firewall.status",
		"permissions.status", "permissions.list_root_access",
		"permissions.discover_scope", "permissions.list_sensitive_access":
		return annotations(true, false, true, false), true
	case "network.check":
		// Read-only from the VPS perspective, but deliberately reaches a
		// policy-authorized network destination.
		return annotations(true, false, true, true), true
	}

	// Permission requests/revocations change Portico authority metadata, not
	// application data. Broker approval and revocation remain authoritative.
	switch name {
	case "permissions.request_root_access", "permissions.request_sensitive_access",
		"permissions.request_elevation",
		"permissions.revoke_root_access", "permissions.revoke_sensitive_access":
		return annotations(false, false, true, false), true
	}

	// Additive local mutation that does not intentionally overwrite existing
	// application content.
	if name == "file.mkdir" {
		return annotations(false, false, true, false), true
	}

	// Filesystem mutations can overwrite, move, delete or change metadata.
	switch name {
	case "file.write_test", "file.write", "file.patch", "file.copy",
		"file.move", "file.remove", "file.chmod", "file.chown":
		return annotations(false, true, false, false), true
	}

	// Starting arbitrary commands is consequential and may contact the outside
	// world when the server-side network policy explicitly allows it.
	switch name {
	case "shell.exec", "job.start", "shell.exec_admin":
		return annotations(false, true, false, true), true
	case "job.cancel":
		return annotations(false, true, true, false), true
	}

	if strings.HasPrefix(name, "service.") {
		switch strings.TrimPrefix(name, "service.") {
		case "start", "stop", "restart", "reload", "enable", "disable":
			return annotations(false, true, false, false), true
		}
	}

	if name == "docker.action" {
		return annotations(false, true, false, false), true
	}

	if strings.HasPrefix(name, "compose.") {
		switch strings.TrimPrefix(name, "compose.") {
		case "pull":
			// Pull changes local image state and contacts an external registry.
			return annotations(false, true, false, true), true
		case "up":
			// Up may pull/build and may cause workloads to contact external
			// services. Conservatively advertise open-world behavior.
			return annotations(false, true, false, true), true
		case "down":
			return annotations(false, true, false, false), true
		}
	}

	if strings.HasPrefix(name, "package.") {
		switch strings.TrimPrefix(name, "package.") {
		case "update", "install":
			return annotations(false, true, false, true), true
		case "remove":
			return annotations(false, true, false, false), true
		}
	}

	if strings.HasPrefix(name, "user.") {
		switch strings.TrimPrefix(name, "user.") {
		case "add", "delete", "lock", "unlock":
			return annotations(false, true, false, false), true
		}
	}

	if strings.HasPrefix(name, "group.") {
		switch strings.TrimPrefix(name, "group.") {
		case "add", "delete":
			return annotations(false, true, false, false), true
		}
	}

	if name == "firewall.action" {
		return annotations(false, true, false, false), true
	}

	return nil, false
}

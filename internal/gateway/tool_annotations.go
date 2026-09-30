package gateway

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolSafety centralizes the MCP client-facing safety hints for every
// model-visible Portico tool. These annotations improve host-native UX and
// confirmation behavior; Broker policy remains the authoritative boundary.
type toolSafety struct {
	readOnly    bool
	destructive bool
	idempotent  bool
	openWorld   bool
}

func safety(readOnly, destructive, idempotent, openWorld bool) toolSafety {
	return toolSafety{
		readOnly: readOnly, destructive: destructive,
		idempotent: idempotent, openWorld: openWorld,
	}
}

func toolSafetyFor(name string) (toolSafety, bool) {
	switch name {
	// Read-only local inspection.
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
		"permissions.discover_scope",
		"permissions.list_root_access", "permissions.list_sensitive_access",
		"permissions.status":
		return safety(true, false, true, false), true

	// Read-only request that deliberately reaches an external destination.
	case "network.check":
		return safety(true, false, true, true), true

	// Filesystem changes.
	case "file.mkdir":
		return safety(false, false, true, false), true
	case "file.write_test", "file.write", "file.copy", "file.chmod", "file.chown":
		return safety(false, true, true, false), true
	case "file.patch", "file.move", "file.remove":
		return safety(false, true, false, false), true

	// systemd actions.
	case "service.start", "service.enable":
		return safety(false, false, true, false), true
	case "service.stop", "service.disable":
		return safety(false, true, true, false), true
	case "service.restart", "service.reload":
		return safety(false, true, false, false), true

	// Docker / Compose mutations.
	case "docker.action":
		return safety(false, true, false, false), true
	case "compose.pull":
		return safety(false, false, true, true), true
	case "compose.up":
		return safety(false, true, false, true), true
	case "compose.down":
		return safety(false, true, true, false), true

	// Arbitrary confined command execution can mutate the delegated project and,
	// when policy explicitly permits egress, can interact with external systems.
	case "shell.exec", "job.start", "shell.exec_admin":
		return safety(false, true, false, true), true
	case "job.cancel":
		return safety(false, true, true, false), true

	// Host package actions.
	case "package.update":
		return safety(false, false, true, true), true
	case "package.install", "package.remove":
		return safety(false, true, false, true), true

	// Identity / firewall mutations.
	case "user.add", "user.delete", "user.lock", "user.unlock",
		"group.add", "group.delete",
		"firewall.action":
		return safety(false, true, false, false), true

	// Authority requests and revocations alter Portico authority, not user data.
	// The Broker remains authoritative and approval requests still require a
	// separate human decision where applicable.
	case "permissions.request_root_access",
		"permissions.revoke_root_access",
		"permissions.request_sensitive_access",
		"permissions.revoke_sensitive_access",
		"permissions.request_elevation":
		return safety(false, false, true, false), true
	default:
		return toolSafety{}, false
	}
}

func annotatedTool(name, description string) *mcp.Tool {
	s, ok := toolSafetyFor(name)
	if !ok {
		panic(fmt.Sprintf("Portico tool %q has no explicit MCP safety annotation classification", name))
	}
	return &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    s.readOnly,
			DestructiveHint: boolPtr(s.destructive),
			IdempotentHint:  s.idempotent,
			OpenWorldHint:   boolPtr(s.openWorld),
		},
	}
}

package gateway

import "testing"

func TestPublicToolAnnotationCatalog(t *testing.T) {
	tests := []struct {
		name                    string
		readOnly, destructive   bool
		idempotent, openWorld   bool
	}{
		{"system.info", true, false, true, false},
		{"system.health", true, false, true, false},
		{"system.disk", true, false, true, false},
		{"system.memory", true, false, true, false},
		{"file.read_test", true, false, true, false},
		{"file.write_test", false, true, true, false},
		{"file.read", true, false, true, false},
		{"file.write", false, true, true, false},
		{"file.mkdir", false, false, true, false},
		{"file.list", true, false, true, false},
		{"file.stat", true, false, true, false},
		{"file.hash", true, false, true, false},
		{"file.patch", false, true, false, false},
		{"file.copy", false, true, true, false},
		{"file.move", false, true, false, false},
		{"file.remove", false, true, false, false},
		{"file.chmod", false, true, true, false},
		{"file.chown", false, true, true, false},
		{"service.list", true, false, true, false},
		{"service.status", true, false, true, false},
		{"service.logs", true, false, true, false},
		{"service.start", false, false, true, false},
		{"service.stop", false, true, true, false},
		{"service.restart", false, true, false, false},
		{"service.reload", false, true, false, false},
		{"service.enable", false, false, true, false},
		{"service.disable", false, true, true, false},
		{"process.list", true, false, true, false},
		{"process.inspect", true, false, true, false},
		{"network.listen", true, false, true, false},
		{"network.check", true, false, true, true},
		{"docker.list", true, false, true, false},
		{"docker.inspect", true, false, true, false},
		{"docker.logs", true, false, true, false},
		{"docker.action", false, true, false, false},
		{"compose.validate", true, false, true, false},
		{"compose.pull", false, false, true, true},
		{"compose.up", false, true, false, true},
		{"compose.down", false, true, true, false},
		{"shell.exec", false, true, false, true},
		{"job.start", false, true, false, true},
		{"shell.exec_admin", false, true, false, true},
		{"job.status", true, false, true, false},
		{"job.tail", true, false, true, false},
		{"job.cancel", false, true, true, false},
		{"package.list", true, false, true, false},
		{"package.update", false, false, true, true},
		{"package.install", false, true, false, true},
		{"package.remove", false, true, false, true},
		{"user.list", true, false, true, false},
		{"user.inspect", true, false, true, false},
		{"user.add", false, true, false, false},
		{"user.delete", false, true, false, false},
		{"user.lock", false, true, false, false},
		{"user.unlock", false, true, false, false},
		{"group.list", true, false, true, false},
		{"group.inspect", true, false, true, false},
		{"group.add", false, true, false, false},
		{"group.delete", false, true, false, false},
		{"firewall.status", true, false, true, false},
		{"firewall.action", false, true, false, false},
		{"permissions.discover_scope", true, false, true, false},
		{"permissions.request_root_access", false, false, true, false},
		{"permissions.revoke_root_access", false, false, true, false},
		{"permissions.list_root_access", true, false, true, false},
		{"permissions.request_sensitive_access", false, false, true, false},
		{"permissions.revoke_sensitive_access", false, false, true, false},
		{"permissions.list_sensitive_access", true, false, true, false},
		{"permissions.request_elevation", false, false, true, false},
		{"permissions.status", true, false, true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toolSafetyFor(tc.name)
			if !ok {
				t.Fatalf("tool %q has no annotation classification", tc.name)
			}
			if got.readOnly != tc.readOnly ||
				got.destructive != tc.destructive ||
				got.idempotent != tc.idempotent ||
				got.openWorld != tc.openWorld {
				t.Fatalf("tool %q annotations = %+v; want readOnly=%v destructive=%v idempotent=%v openWorld=%v",
					tc.name, got, tc.readOnly, tc.destructive, tc.idempotent, tc.openWorld)
			}
			tool := annotatedTool(tc.name, "test")
			if tool.Annotations == nil ||
				tool.Annotations.ReadOnlyHint != tc.readOnly ||
				tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != tc.destructive ||
				tool.Annotations.IdempotentHint != tc.idempotent ||
				tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint != tc.openWorld {
				t.Fatalf("tool %q did not expose the classified annotations: %+v", tc.name, tool.Annotations)
			}
		})
	}
}

func TestUnknownToolClassificationFailsClosed(t *testing.T) {
	if _, ok := toolSafetyFor("future.unclassified"); ok {
		t.Fatal("unknown tool unexpectedly classified")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("annotatedTool must panic for an unclassified public tool")
		}
	}()
	_ = annotatedTool("future.unclassified", "must fail")
}

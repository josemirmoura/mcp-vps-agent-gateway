package broker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sensitive"
)

func (b *Broker) requireSensitiveSubtreeWork(ctx context.Context, subject, target string, roots []string) error {
	paths, err := sensitive.ProtectedAliases(
		os.Getenv("VPS_AGENT_HOST_ROOT"),
		roots,
		sensitive.DefaultScanLimit,
	)
	if err != nil {
		return err
	}
	access, err := b.activeSensitiveAccess(ctx, subject)
	if err != nil {
		return err
	}
	target = filepath.Clean(target)
	for _, path := range paths {
		if !pathWithinRoot(target, path) {
			continue
		}
		if access[filepath.Clean(path)] != "work" {
			return fmt.Errorf("protected path %s requires a separate work approval before the tree operation", path)
		}
	}
	return nil
}

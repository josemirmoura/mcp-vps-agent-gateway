package cloudnode

import (
	"errors"
	"os"
	"path/filepath"
)

// QuarantineMarkerPath locates a node-local persistent stop marker next to the
// private identity state. It contains no keys, tokens, or task payloads.
func QuarantineMarkerPath(statePath string) string {
	return statePath + ".quarantine"
}

// CheckQuarantine fails closed if a previous connector invocation could not
// confirm that the Broker stopped. Operators must inspect Broker audit and
// deliberately remove the marker before restoring Cloud task polling.
func CheckQuarantine(statePath string) error {
	if statePath == "" {
		return errors.New("node identity state path is required")
	}
	_, err := os.Lstat(QuarantineMarkerPath(statePath))
	if err == nil {
		return errors.New("Cloud connector quarantined after unconfirmed Broker cancellation; operator review required")
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// MarkQuarantine stores a persistent marker before the connector exits.
// O_EXCL prevents accidental overwrite or following an existing symlink.
func MarkQuarantine(statePath string) error {
	if statePath == "" {
		return errors.New("node identity state path is required")
	}
	path := QuarantineMarkerPath(statePath)
	dir := filepath.Dir(path)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil // Already quarantined, including a pre-existing marker.
	}
	if err != nil {
		return err
	}
	if _, err := file.WriteString("broker_cancellation_unconfirmed\n"); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	parent, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}

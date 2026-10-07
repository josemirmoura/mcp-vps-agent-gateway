package cloudnode

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateIdentity() (Identity, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey),
		PublicKey:  base64.RawURLEncoding.EncodeToString(publicKey),
	}, nil
}

func (identity Identity) Private() (ed25519.PrivateKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(identity.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key length")
	}
	privateKey := ed25519.PrivateKey(raw)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if base64.RawURLEncoding.EncodeToString(publicKey) != identity.PublicKey {
		return nil, errors.New("private/public key mismatch")
	}
	return privateKey, nil
}

func ValidateIdentity(identity Identity) error {
	if identity.NodeID == "" || identity.WorkspaceID == "" || identity.PublicKey == "" || identity.Fingerprint == "" {
		return errors.New("identity is incomplete")
	}
	_, err := identity.Private()
	return err
}

func ValidateState(state NodeState) error {
	if _, err := state.Identity.Private(); err != nil {
		return err
	}

	if state.Identity.NodeID == "" {
		if state.Enrollment == nil ||
			state.Enrollment.Token == "" ||
			state.Enrollment.NodeName == "" ||
			state.Enrollment.Platform == "" {
			return errors.New("pending enrollment state is incomplete")
		}
		if state.Identity.WorkspaceID != "" || state.Identity.Fingerprint != "" {
			return errors.New("pending enrollment contains partial server identity")
		}
		return nil
	}

	if state.Enrollment != nil {
		return errors.New("enrolled identity must not retain the bootstrap token")
	}
	return ValidateIdentity(state.Identity)
}

func LoadState(path string) (NodeState, error) {
	if path == "" {
		return NodeState{}, errors.New("identity path is required")
	}

	info, err := os.Lstat(path)
	if err != nil {
		return NodeState{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return NodeState{}, errors.New("identity path must not be a symlink")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return NodeState{}, errors.New("identity file permissions are too broad")
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return NodeState{}, err
	}
	defer root.Close()

	data, err := root.ReadFile(base)
	if err != nil {
		return NodeState{}, err
	}
	if len(data) > 32<<10 {
		return NodeState{}, errors.New("identity file is unexpectedly large")
	}

	var state NodeState
	if err := json.Unmarshal(data, &state); err != nil {
		return NodeState{}, err
	}
	if err := ValidateState(state); err != nil {
		return NodeState{}, err
	}
	return state, nil
}

func SaveState(path string, state NodeState) error {
	if path == "" {
		return errors.New("identity path is required")
	}
	if err := ValidateState(state); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '
')

	temp, err := os.CreateTemp(dir, ".identity-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}

	parent, err := os.Open(dir)
	if err == nil {
		_ = parent.Sync()
		_ = parent.Close()
	}
	return nil
}

package securefs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxBytes int64 = 1 << 20

type Manager struct {
	readRoots  []string
	writeRoots []string
	maxBytes   int64
}

func New(readRoots, writeRoots []string, maxBytes int64) (*Manager, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	normalize := func(in []string) ([]string, error) {
		out := make([]string, 0, len(in))
		for _, r := range in {
			abs, err := filepath.Abs(r)
			if err != nil {
				return nil, err
			}
			out = append(out, filepath.Clean(abs))
		}
		return out, nil
	}
	rr, err := normalize(readRoots)
	if err != nil {
		return nil, err
	}
	wr, err := normalize(writeRoots)
	if err != nil {
		return nil, err
	}
	return &Manager{readRoots: rr, writeRoots: wr, maxBytes: maxBytes}, nil
}

func rejectTraversal(p string) error {
	if p == "" || !filepath.IsAbs(p) {
		return errors.New("path must be absolute")
	}
	for _, part := range strings.FieldsFunc(filepath.ToSlash(p), func(r rune) bool { return r == '/' }) {
		if part == ".." {
			return errors.New("path traversal is forbidden")
		}
	}
	return nil
}

func selectRoot(roots []string, target string) (root, rel string, err error) {
	if err := rejectTraversal(target); err != nil {
		return "", "", err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	abs = filepath.Clean(abs)
	for _, candidate := range roots {
		r, err := filepath.Rel(candidate, abs)
		if err != nil || r == "." {
			if err == nil && r == "." {
				return candidate, ".", nil
			}
			continue
		}
		if r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) || filepath.IsAbs(r) {
			continue
		}
		return candidate, r, nil
	}
	return "", "", errors.New("path outside authorized roots")
}


func selectAuthorizedRoot(roots []string, target string) (string, error) {
	if err := rejectTraversal(target); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	for _, candidate := range roots {
		rel, err := filepath.Rel(candidate, abs)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
			return candidate, nil
		}
	}
	return "", errors.New("path outside authorized roots")
}

func nearestExistingDir(p string) (string, error) {
	cur := filepath.Clean(p)
	for {
		info, err := os.Lstat(cur)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("authorized path ancestor is a symlink: %s", cur)
			}
			if !info.IsDir() {
				return "", fmt.Errorf("authorized path ancestor is not a directory: %s", cur)
			}
			resolved, err := filepath.EvalSymlinks(cur)
			if err != nil {
				return "", err
			}
			if filepath.Clean(resolved) != cur {
				return "", fmt.Errorf("authorized path ancestor resolves through symlink: %s", cur)
			}
			return cur, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no existing directory ancestor for %s", p)
		}
		cur = parent
	}
}

// MkdirAll creates an authorized directory tree while rejecting traversal and
// symlink components. It can create the configured writable root itself when
// that root does not yet exist.
func (m *Manager) MkdirAll(dirname string, perm os.FileMode) error {
	allowedRoot, err := selectAuthorizedRoot(m.writeRoots, dirname)
	if err != nil {
		return err
	}
	target, err := filepath.Abs(dirname)
	if err != nil {
		return err
	}
	target = filepath.Clean(target)

	anchor, err := nearestExistingDir(allowedRoot)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(anchor, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("directory creation escaped authorized anchor")
	}

	root, err := os.OpenRoot(anchor)
	if err != nil {
		return err
	}
	defer root.Close()

	current := "."
	for _, part := range strings.FieldsFunc(filepath.ToSlash(rel), func(r rune) bool { return r == '/' }) {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return errors.New("path traversal is forbidden")
		}
		if current == "." {
			current = part
		} else {
			current = filepath.Join(current, part)
		}
		info, err := root.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlink component: %s", current)
			}
			if !info.IsDir() {
				return fmt.Errorf("path component is not a directory: %s", current)
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := root.Mkdir(current, perm.Perm()); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ReadFile(filename string) ([]byte, error) {
	rootPath, rel, err := selectRoot(m.readRoots, filename)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lr := io.LimitReader(f, m.maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > m.maxBytes {
		return nil, fmt.Errorf("file exceeds %d-byte limit", m.maxBytes)
	}
	return data, nil
}

func (m *Manager) WriteFileAtomic(filename string, data []byte) error {
	if int64(len(data)) > m.maxBytes {
		return fmt.Errorf("content exceeds %d-byte limit", m.maxBytes)
	}
	rootPath, rel, err := selectRoot(m.writeRoots, filename)
	if err != nil {
		return err
	}
	if rel == "." {
		return errors.New("cannot overwrite root directory")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()

	dir := filepath.Dir(rel)
	base := filepath.Base(rel)
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	tmpBase := "." + base + ".tmp-" + hex.EncodeToString(nonce[:])
	tmpRel := tmpBase
	if dir != "." {
		tmpRel = filepath.Join(dir, tmpBase)
	}

	f, err := root.OpenFile(tmpRel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		_ = f.Close()
		if cleanup {
			_ = root.Remove(tmpRel)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := root.Rename(tmpRel, rel); err != nil {
		return err
	}
	cleanup = false
	return nil
}

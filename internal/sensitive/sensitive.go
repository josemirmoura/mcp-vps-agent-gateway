package sensitive

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const DefaultScanLimit = 50000

var ErrScanLimit = errors.New("sensitive-path scan exceeded safety limit")

func ProtectedName(path string) bool {
	name := filepath.Base(filepath.Clean(path))
	switch name {
	case ".env.example", ".env.sample", ".env.template":
		return false
	}
	return name == ".env" || strings.HasPrefix(name, ".env.")
}

func hostPath(hostRoot, canonical string) string {
	clean := filepath.Clean(canonical)
	if hostRoot == "" || hostRoot == "/" {
		return clean
	}
	if clean == "/" {
		return filepath.Clean(hostRoot)
	}
	return filepath.Join(filepath.Clean(hostRoot), strings.TrimPrefix(clean, string(filepath.Separator)))
}

type inodeKey struct {
	dev uint64
	ino uint64
}

func inode(info fs.FileInfo) (inodeKey, uint64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return inodeKey{}, 0, false
	}
	return inodeKey{dev: uint64(st.Dev), ino: uint64(st.Ino)}, uint64(st.Nlink), true
}

func IsProtected(hostRoot, target string, _ []string, _ int) (bool, error) {
	target = filepath.Clean(target)
	if ProtectedName(target) {
		return true, nil
	}
	info, err := os.Lstat(hostPath(hostRoot, target))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	_, links, ok := inode(info)
	// A multiply-linked inode is ambiguous: another name may be a protected
	// secret outside the currently delegated tree. Treat it as protected
	// rather than trying to infer provenance from a partial filesystem view.
	return ok && links > 1, nil
}

func ProtectedAliases(hostRoot string, roots []string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = DefaultScanLimit
	}
	outSet := map[string]struct{}{}
	seen := 0
	for _, root := range roots {
		root = filepath.Clean(root)
		host := hostPath(hostRoot, root)
		err := filepath.WalkDir(host, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			seen++
			if seen > limit {
				return ErrScanLimit
			}
			if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			if !fi.Mode().IsRegular() {
				return nil
			}
			_, links, ok := inode(fi)
			if !ok {
				return nil
			}
			rel, err := filepath.Rel(host, path)
			if err != nil {
				return err
			}
			canonical := root
			if rel != "." {
				canonical = filepath.Join(root, rel)
			}
			if ProtectedName(canonical) || links > 1 {
				outSet[filepath.Clean(canonical)] = struct{}{}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(outSet))
	for p := range outSet {
		out = append(out, p)
	}
	return out, nil
}

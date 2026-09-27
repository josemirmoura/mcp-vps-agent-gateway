package securefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadWriteAndTraversal(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(root, "hello.txt")
	if err := m.WriteFileAtomic(target, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	got, err := m.ReadFile(target)
	if err != nil || string(got) != "hello" {
		t.Fatalf("read got %q err=%v", string(got), err)
	}

	traversal := filepath.Join(root, "..", filepath.Base(outside), "secret")
	if _, err := m.ReadFile(traversal); err == nil {
		t.Fatal("expected traversal denial")
	}
}

func TestSymlinkEscapeDenied(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("classified"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	m, err := New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReadFile(filepath.Join(root, "escape")); err == nil {
		t.Fatal("expected os.Root to reject symlink escape")
	}
}

func TestPrefixConfusionDenied(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "app")
	confusing := filepath.Join(base, "application")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(confusing, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confusing, "secret"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, _ := New([]string{root}, []string{root}, 1024)
	if _, err := m.ReadFile(filepath.Join(confusing, "secret")); err == nil {
		t.Fatal("prefix-confused path must be denied")
	}
}


func TestMkdirAllCanCreateConfiguredRoot(t *testing.T) {
	base := t.TempDir()
	targetRoot := filepath.Join(base, "new-root")
	m, err := New([]string{targetRoot}, []string{targetRoot}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.MkdirAll(targetRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(targetRoot)
	if err != nil || !info.IsDir() {
		t.Fatalf("configured root was not created: info=%v err=%v", info, err)
	}
	if err := m.WriteFileAtomic(filepath.Join(targetRoot, "index.html"), []byte("<3")); err != nil {
		t.Fatal(err)
	}
}

func TestMkdirAllRejectsOutsideAndSymlink(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "allowed")
	if err := os.MkdirAll(root, 0o750); err != nil {
		t.Fatal(err)
	}
	m, err := New([]string{root}, []string{root}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.MkdirAll(filepath.Join(base, "outside"), 0o750); err == nil {
		t.Fatal("outside directory creation was allowed")
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := m.MkdirAll(filepath.Join(root, "link", "child"), 0o750); err == nil {
		t.Fatal("symlink directory component was followed")
	}
}


func TestCompleteFilesystemToolbox(t *testing.T) {
	root := t.TempDir()
	m, err := New([]string{root}, []string{root}, 4096)
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, "a", "b")
	if err := m.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file.txt")
	if err := m.WriteFileAtomic(file, []byte("alpha beta gamma")); err != nil {
		t.Fatal(err)
	}

	entries, err := m.List(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "file.txt" {
		t.Fatalf("unexpected list: %+v", entries)
	}

	st, err := m.Stat(file, false)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size != int64(len("alpha beta gamma")) || st.IsDir {
		t.Fatalf("unexpected stat: %+v", st)
	}

	hash1, err := m.Hash(file)
	if err != nil || len(hash1) != 64 {
		t.Fatalf("hash=%q err=%v", hash1, err)
	}

	hash2, err := m.Patch(file, "beta", "BETA", hash1)
	if err != nil || hash2 == hash1 {
		t.Fatalf("patch hash=%q err=%v", hash2, err)
	}
	data, _ := m.ReadFile(file)
	if string(data) != "alpha BETA gamma" {
		t.Fatalf("patch result=%q", data)
	}

	copyPath := filepath.Join(dir, "copy.txt")
	if err := m.CopyFile(file, copyPath); err != nil {
		t.Fatal(err)
	}
	movePath := filepath.Join(dir, "moved.txt")
	if err := m.Move(copyPath, movePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(copyPath); !os.IsNotExist(err) {
		t.Fatalf("copy source still exists after move: %v", err)
	}

	if err := m.Chmod(movePath, 0o777); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(movePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o777 {
		t.Fatalf("chmod got %o", info.Mode().Perm())
	}

	if err := m.Remove(movePath, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove(filepath.Join(root, "a"), true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatalf("recursive removal failed: %v", err)
	}
}

func TestPatchPreconditionAndAmbiguity(t *testing.T) {
	root := t.TempDir()
	m, _ := New([]string{root}, []string{root}, 4096)
	file := filepath.Join(root, "x.txt")
	_ = m.WriteFileAtomic(file, []byte("same same"))
	if _, err := m.Patch(file, "same", "new", ""); err == nil {
		t.Fatal("ambiguous patch was allowed")
	}
	if _, err := m.Patch(file, "same same", "new", strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong sha precondition was allowed")
	}
}

func TestRecursiveRemoveDoesNotFollowSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	m, _ := New([]string{root}, []string{root}, 4096)
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(tree, 0o750); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(tree, "outside-link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := m.Remove(tree, true); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(secret); err != nil || string(got) != "keep" {
		t.Fatalf("recursive delete escaped through symlink: got=%q err=%v", got, err)
	}
}

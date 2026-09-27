package securefs

import (
	"os"
	"path/filepath"
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

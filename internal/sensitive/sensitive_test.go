package sensitive

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedName(t *testing.T) {
	for _, p := range []string{"/srv/app/.env", "/srv/app/.env.local", "/srv/app/sub/.env.production"} {
		if !ProtectedName(p) {
			t.Fatalf("expected protected: %s", p)
		}
	}
	for _, p := range []string{"/srv/app/.env.example", "/srv/app/.env.sample", "/srv/app/.env.template", "/srv/app/env"} {
		if ProtectedName(p) {
			t.Fatalf("expected readable template/ordinary file: %s", p)
		}
	}
}

func TestHardlinkAliasIsProtected(t *testing.T) {
	root := t.TempDir()
	env := filepath.Join(root, ".env")
	alias := filepath.Join(root, "innocent.txt")
	if err := os.WriteFile(env, []byte("private-value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(env, alias); err != nil {
		t.Fatal(err)
	}
	ok, err := IsProtected("", alias, []string{root}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("hardlink alias of .env must be protected")
	}
	aliases, err := ProtectedAliases("", []string{root}, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundEnv, foundAlias := false, false
	for _, p := range aliases {
		if p == env {
			foundEnv = true
		}
		if p == alias {
			foundAlias = true
		}
	}
	if !foundEnv || !foundAlias {
		t.Fatalf("expected .env and hardlink alias, got %#v", aliases)
	}
}

func TestScanLimitFailsClosed(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(root, string(rune('a'+i))), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := ProtectedAliases("", []string{root}, 2)
	if !errors.Is(err, ErrScanLimit) {
		t.Fatalf("expected ErrScanLimit, got %v", err)
	}
}

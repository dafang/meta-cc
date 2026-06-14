package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yaleh/meta-cc/internal/locator"
)

func TestProviderAvailability(t *testing.T) {
	root := t.TempDir()
	loc := locator.NewCodexLocator()
	_ = loc

	t.Setenv("META_CC_CODEX_ROOT", root)
	p := NewProvider(locator.NewCodexLocator())
	if p.IsAvailable(context.Background()) {
		t.Fatalf("expected unavailable")
	}

	dbFile := filepath.Join(root, "state_5.sqlite")
	if err := os.WriteFile(dbFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !p.IsAvailable(context.Background()) {
		t.Fatalf("expected available")
	}
}

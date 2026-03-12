package observability_test

import (
	"testing"

	"github.com/yaleh/meta-cc/internal/mcp/observability"
)

// TestInitTracing tests the InitTracing function
func TestInitTracing(t *testing.T) {
	cleanup, err := observability.InitTracing()
	if err != nil {
		t.Fatalf("InitTracing failed: %v", err)
	}
	defer cleanup()
}

package pipeline_test

import (
	"testing"

	"github.com/yaleh/meta-cc/internal/query/pipeline"
	"github.com/yaleh/meta-cc/internal/types"
)

func makeEntries(roles ...string) []types.SessionEntry {
	entries := make([]types.SessionEntry, len(roles))
	for i, role := range roles {
		entries[i] = types.SessionEntry{
			Type:      role,
			UUID:      "uuid-" + role,
			SessionID: "session-1",
			Timestamp: "2025-01-01T00:00:00Z",
		}
		if role == "user" || role == "assistant" {
			entries[i].Message = &types.Message{Role: role}
		}
	}
	return entries
}

func TestSelectResource_Entries(t *testing.T) {
	entries := makeEntries("user", "assistant")
	result, err := pipeline.SelectResource(entries, "entries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := result.([]types.SessionEntry)
	if !ok {
		t.Fatalf("expected []types.SessionEntry, got %T", result)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestSelectResource_Tools(t *testing.T) {
	entries := makeEntries("user")
	result, err := pipeline.SelectResource(entries, "tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestSelectResource_UnknownType(t *testing.T) {
	_, err := pipeline.SelectResource(nil, "unknown")
	if err == nil {
		t.Fatal("expected error for unknown resource type")
	}
}

func TestApplyFilter_EmptyFilterPassesThrough(t *testing.T) {
	entries := makeEntries("user", "assistant")
	result := pipeline.ApplyFilter(entries, pipeline.FilterSpec{})
	got, ok := result.([]types.SessionEntry)
	if !ok {
		t.Fatalf("expected []types.SessionEntry, got %T", result)
	}
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestApplyFilter_ByType(t *testing.T) {
	entries := makeEntries("user", "assistant", "user")
	result := pipeline.ApplyFilter(entries, pipeline.FilterSpec{Type: "user"})
	got := result.([]types.SessionEntry)
	if len(got) != 2 {
		t.Errorf("expected 2 user entries, got %d", len(got))
	}
}

func TestApplyAggregate_EmptyAggregatePassesThrough(t *testing.T) {
	entries := makeEntries("user", "assistant")
	result := pipeline.ApplyAggregate(entries, pipeline.AggregateSpec{})
	got, ok := result.([]types.SessionEntry)
	if !ok {
		t.Fatalf("expected []types.SessionEntry, got %T", result)
	}
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestApplyAggregate_Count(t *testing.T) {
	entries := makeEntries("user", "assistant", "user")
	result := pipeline.ApplyAggregate(entries, pipeline.AggregateSpec{Function: "count"})
	got, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatalf("expected []map, got %T", result)
	}
	if len(got) != 1 || got[0]["count"].(int) != 3 {
		t.Errorf("expected count=3, got %v", got)
	}
}

func TestValidateQueryParams_ValidResource(t *testing.T) {
	params := pipeline.QueryParams{Resource: "messages"}
	if err := pipeline.ValidateQueryParams(params); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateQueryParams_InvalidResource(t *testing.T) {
	params := pipeline.QueryParams{Resource: "invalid"}
	if err := pipeline.ValidateQueryParams(params); err == nil {
		t.Fatal("expected error for invalid resource")
	}
}

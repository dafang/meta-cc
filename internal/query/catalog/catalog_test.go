package catalog_test

import (
	"testing"

	"github.com/yaleh/meta-cc/internal/query/catalog"
)

func TestLoadTemplates_ReturnsMap(t *testing.T) {
	templates, err := catalog.LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates returned error: %v", err)
	}
	// Templates dir may or may not exist depending on cwd; we just verify no crash
	if templates == nil {
		t.Fatal("expected non-nil map")
	}
}

func TestQueryTemplate_Fields(t *testing.T) {
	tmpl := catalog.QueryTemplate{
		Name:        "test",
		Description: "A test template",
		Category:    "test",
		Filter:      `.[] | select(.type == "user")`,
	}
	if tmpl.Name != "test" {
		t.Errorf("unexpected Name: %v", tmpl.Name)
	}
	if tmpl.Description != "A test template" {
		t.Errorf("unexpected Description: %v", tmpl.Description)
	}
}

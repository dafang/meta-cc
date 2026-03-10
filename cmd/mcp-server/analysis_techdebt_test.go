package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaleh/meta-cc/internal/analysis"
)

// TestGetTechDebtToolRegistered verifies get_tech_debt appears in getToolDefinitions()
func TestGetTechDebtToolRegistered(t *testing.T) {
	tools := getToolDefinitions()
	for _, tool := range tools {
		if tool.Name == "get_tech_debt" {
			return
		}
	}
	t.Fatal("get_tech_debt not found in tool definitions")
}

// TestGetTechDebtToolExecution loads test.jsonl and verifies valid JSON output
func TestGetTechDebtToolExecution(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"working_dir": projectPath,
	}

	output, err := analysis.New().GetTechDebt(args)
	require.NoError(t, err, "executeGetTechDebtTool should not return error")
	require.NotEmpty(t, output, "output should not be empty")

	// Verify output is valid JSON with expected fields
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	_, hasMarkers := result["markers"]
	assert.True(t, hasMarkers, "result should have 'markers' field")

	_, hasHotspotFiles := result["hotspot_files"]
	assert.True(t, hasHotspotFiles, "result should have 'hotspot_files' field")

	_, hasOpenIssues := result["open_issues"]
	assert.True(t, hasOpenIssues, "result should have 'open_issues' field")
}

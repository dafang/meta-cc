package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyzeBugsToolRegistered verifies analyze_bugs appears in getToolDefinitions()
func TestAnalyzeBugsToolRegistered(t *testing.T) {
	tools := getToolDefinitions()
	for _, tool := range tools {
		if tool.Name == "analyze_bugs" {
			return
		}
	}
	t.Fatal("analyze_bugs not found in tool definitions")
}

// TestAnalyzeBugsToolExecution loads test.jsonl and verifies valid JSON output
func TestAnalyzeBugsToolExecution(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"working_dir": projectPath,
	}

	output, err := executeAnalyzeBugsTool(nil, args)
	require.NoError(t, err, "executeAnalyzeBugsTool should not return error")
	require.NotEmpty(t, output, "output should not be empty")

	// Verify output is valid JSON with expected fields
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	_, hasPatterns := result["patterns"]
	assert.True(t, hasPatterns, "result should have 'patterns' field")

	_, hasTotalPairs := result["total_pairs"]
	assert.True(t, hasTotalPairs, "result should have 'total_pairs' field")
}

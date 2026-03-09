package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQualityScanToolRegistered verifies quality_scan appears in getToolDefinitions()
func TestQualityScanToolRegistered(t *testing.T) {
	tools := getToolDefinitions()
	found := false
	for _, tool := range tools {
		if tool.Name == "quality_scan" {
			found = true
			break
		}
	}
	assert.True(t, found, "quality_scan tool should be registered in getToolDefinitions()")
}

// TestQualityScanToolExecution calls the handler against test.jsonl and verifies output.
func TestQualityScanToolExecution(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"working_dir": projectPath,
	}

	output, err := executeQualityScanTool(nil, args)
	require.NoError(t, err, "executeQualityScanTool should not return error")
	require.NotEmpty(t, output, "output should not be empty")

	// Verify output is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	// Verify dimensions array exists
	dims, hasDims := result["dimensions"]
	assert.True(t, hasDims, "result should have dimensions field")

	dimsSlice, ok := dims.([]interface{})
	require.True(t, ok, "dimensions should be an array")
	assert.GreaterOrEqual(t, len(dimsSlice), 4, "should have at least 4 dimensions")

	// Verify each dimension has required fields and score in [0,1]
	for i, d := range dimsSlice {
		dim, ok := d.(map[string]interface{})
		require.True(t, ok, "dimension %d should be an object", i)
		assert.Contains(t, dim, "name", "dimension %d should have name", i)
		assert.Contains(t, dim, "score", "dimension %d should have score", i)
		score, ok := dim["score"].(float64)
		require.True(t, ok, "dimension %d score should be a number", i)
		assert.GreaterOrEqual(t, score, 0.0, "dimension %d score should be >= 0", i)
		assert.LessOrEqual(t, score, 1.0, "dimension %d score should be <= 1", i)
	}
}

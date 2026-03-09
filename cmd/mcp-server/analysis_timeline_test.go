package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetTimelineToolRegistered verifies get_timeline appears in getToolDefinitions()
func TestGetTimelineToolRegistered(t *testing.T) {
	tools := getToolDefinitions()
	for _, tool := range tools {
		if tool.Name == "get_timeline" {
			return
		}
	}
	t.Fatal("get_timeline not found in tool definitions")
}

// TestGetTimelineToolExecution loads test.jsonl and verifies output has events and total_span fields
func TestGetTimelineToolExecution(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"working_dir": projectPath,
	}

	output, err := executeGetTimelineTool(nil, args)
	require.NoError(t, err, "executeGetTimelineTool should not return error")
	require.NotEmpty(t, output, "output should not be empty")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	_, hasEvents := result["events"]
	assert.True(t, hasEvents, "result should have events field")

	_, hasTotalSpan := result["total_span"]
	assert.True(t, hasTotalSpan, "result should have total_span field")
}

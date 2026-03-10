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

func TestGetWorkPatternsToolRegistered(t *testing.T) {
	tools := getToolDefinitions()
	for _, tool := range tools {
		if tool.Name == "get_work_patterns" {
			return
		}
	}
	t.Fatal("get_work_patterns not found in tool definitions")
}

func TestGetWorkPatternsToolExecution(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"working_dir": projectPath,
	}

	output, err := analysis.New().GetWorkPatterns(args)
	require.NoError(t, err, "executeGetWorkPatternsTool should not return error")
	require.NotEmpty(t, output, "output should not be empty")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	_, hasToolFrequency := result["tool_frequency"]
	assert.True(t, hasToolFrequency, "result should have tool_frequency field")

	_, hasHourlyActivity := result["hourly_activity"]
	assert.True(t, hasHourlyActivity, "result should have hourly_activity field")
}

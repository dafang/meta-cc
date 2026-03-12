package main

import (
	responsepkg "github.com/yaleh/meta-cc/internal/mcp/response"
)

// FileReference provides metadata about a temporary JSONL file.
type FileReference = responsepkg.FileReference

// generateFileReference creates a FileReference with metadata for a JSONL file.
func generateFileReference(filePath string, data []interface{}) (*FileReference, error) {
	return responsepkg.GenerateFileReference(filePath, data)
}

// extractFields extracts unique field names from JSONL records.
func extractFields(records []interface{}) []string {
	return responsepkg.ExtractFields(records)
}

// generateSummary creates summary statistics for JSONL records.
func generateSummary(records []interface{}) map[string]interface{} {
	return responsepkg.GenerateSummary(records)
}

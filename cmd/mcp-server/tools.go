package main

import (
	"github.com/yaleh/meta-cc/internal/mcp/schema"
	"github.com/yaleh/meta-cc/internal/mcp/tools"
)

// Type aliases so existing code in this package compiles unchanged.
type Tool = schema.Tool
type ToolSchema = schema.ToolSchema
type Property = schema.Property

// StandardToolParameters returns the standard set of parameters for all MCP tools
func StandardToolParameters() map[string]Property {
	return tools.StandardToolParameters()
}

// MergeParameters merges tool-specific params with standard params
func MergeParameters(specific map[string]Property) map[string]Property {
	return tools.MergeParameters(specific)
}

func getToolDefinitions() []Tool {
	return tools.GetToolDefinitions()
}

// toolSchemaIndex caches the mapping from tool name to ToolSchema.
// Tests can reset this to nil to force a rebuild.
var toolSchemaIndex map[string]ToolSchema

// buildToolSchemaIndex builds the index from tool definitions, lazily on first call.
func buildToolSchemaIndex() map[string]ToolSchema {
	if toolSchemaIndex != nil {
		return toolSchemaIndex
	}
	toolSchemaIndex = tools.BuildToolSchemaIndex()
	return toolSchemaIndex
}

// getToolSchemaByName returns the ToolSchema for the named tool, or an error if not found.
func getToolSchemaByName(name string) (ToolSchema, error) {
	return tools.GetToolSchemaByName(buildToolSchemaIndex(), name)
}

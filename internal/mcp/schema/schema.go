package schema

import (
	"fmt"
	"sort"
	"strings"
)

// Tool represents an MCP tool with name, description, and input schema.
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema ToolSchema `json:"inputSchema"`
}

// ToolSchema defines the JSON schema for a tool's input parameters.
type ToolSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property defines a single JSON schema property.
type Property struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Items       *Property `json:"items,omitempty"` // For array types
}

// BuildSchemaIndex builds a map from tool name to ToolSchema from a slice of Tool definitions.
func BuildSchemaIndex(defs []Tool) map[string]ToolSchema {
	index := make(map[string]ToolSchema, len(defs))
	for _, t := range defs {
		index[t.Name] = t.InputSchema
	}
	return index
}

// GetByName returns the ToolSchema for the named tool from the given index,
// or an error if the tool is not found.
func GetByName(index map[string]ToolSchema, name string) (ToolSchema, error) {
	s, ok := index[name]
	if !ok {
		return ToolSchema{}, fmt.Errorf("unknown tool %s: no schema found", name)
	}
	return s, nil
}

// ValidateArgKeys checks that all keys in args are declared in the tool schema.
// Returns an error listing unknown keys and the valid options.
func ValidateArgKeys(args map[string]interface{}, schema ToolSchema) error {
	if len(args) == 0 {
		return nil
	}

	var unknown []string
	for key := range args {
		if _, ok := schema.Properties[key]; !ok {
			unknown = append(unknown, key)
		}
	}

	if len(unknown) == 0 {
		return nil
	}

	// Sort for deterministic error messages
	sort.Strings(unknown)

	var valid []string
	for key := range schema.Properties {
		valid = append(valid, key)
	}
	sort.Strings(valid)

	return fmt.Errorf("unknown parameter(s): %s; valid parameters are: %s",
		strings.Join(unknown, ", "),
		strings.Join(valid, ", "))
}

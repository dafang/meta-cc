package main

import (
	"github.com/yaleh/meta-cc/internal/config"
	responsepkg "github.com/yaleh/meta-cc/internal/mcp/response"
)

const (
	DefaultInlineThresholdBytes = responsepkg.DefaultInlineThresholdBytes
	OutputModeInline            = responsepkg.OutputModeInline
	OutputModeFileRef           = responsepkg.OutputModeFileRef
)

// OutputModeConfig holds configuration for output mode selection
type OutputModeConfig = responsepkg.OutputModeConfig

// DefaultOutputModeConfig returns the default configuration
func DefaultOutputModeConfig() *OutputModeConfig {
	return responsepkg.DefaultOutputModeConfig()
}

// calculateOutputSize measures the byte size of data when serialized to JSONL format.
func calculateOutputSize(data []interface{}) int {
	return responsepkg.CalculateOutputSize(data)
}

// selectOutputMode determines whether to use inline or file_ref mode based on data size.
func selectOutputMode(size int, explicitMode string) string {
	return responsepkg.SelectOutputMode(size, explicitMode)
}

// selectOutputModeWithConfig is the same as selectOutputMode but allows custom configuration.
func selectOutputModeWithConfig(size int, explicitMode string, config *OutputModeConfig) string {
	return responsepkg.SelectOutputModeWithConfig(size, explicitMode, config)
}

// getOutputModeConfig returns output mode configuration from centralized config and parameters.
func getOutputModeConfig(globalCfg *config.Config, params map[string]interface{}) *OutputModeConfig {
	return responsepkg.GetOutputModeConfig(globalCfg, params)
}

package main

import (
	"github.com/yaleh/meta-cc/internal/analysis"
	"github.com/yaleh/meta-cc/internal/config"
)

// executeAnalyzeBugsTool implements the analyze_bugs MCP tool.
func executeAnalyzeBugsTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	return analysis.New(cfg).AnalyzeBugs(args)
}

// executeQualityScanTool implements the quality_scan MCP tool.
func executeQualityScanTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	return analysis.New(cfg).QualityScan(args)
}

// executeGetWorkPatternsTool implements the get_work_patterns MCP tool.
func executeGetWorkPatternsTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	return analysis.New(cfg).GetWorkPatterns(args)
}

// executeGetTimelineTool implements the get_timeline MCP tool.
func executeGetTimelineTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	return analysis.New(cfg).GetTimeline(args)
}

// executeGetTechDebtTool implements the get_tech_debt MCP tool.
func executeGetTechDebtTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	return analysis.New(cfg).GetTechDebt(args)
}

// executeAnalyzeErrorsTool implements the analyze_errors MCP tool.
func executeAnalyzeErrorsTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	return analysis.New(cfg).AnalyzeErrors(args)
}

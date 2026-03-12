// Package analyzer provides the business-logic layer of analysis.
//
// Architecture:
//
//	internal/analysis  (facade)
//	   ↓ injects via Analyzers struct
//	internal/analyzer.DefaultAnalyzer  (interface adapter)
//	   ↓ delegates to
//	internal/analyzer.<function>()     (pure functions, no I/O)
//
// Domain interfaces (BugAnalyzer, ErrorAnalyzer, QualityScanner,
// WorkPatternsAnalyzer, TimelineAnalyzer, TechDebtAnalyzer) use
// []parser.SessionEntry and []parser.ToolCall — these are type aliases
// for []types.SessionEntry and []types.ToolCall defined in
// internal/parser/aliases.go. No type conversion occurs at the boundary.
//
// DefaultAnalyzer is a thin adapter that allows cmd/mcp-server tests
// to substitute mock implementations via the six interface types.
// Business logic lives exclusively in the package-level functions;
// DefaultAnalyzer adds no logic of its own.
package analyzer

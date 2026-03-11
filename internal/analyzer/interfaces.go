package analyzer

import "github.com/yaleh/meta-cc/internal/parser"

// BugAnalyzer analyzes session entries for bugs.
type BugAnalyzer interface {
	AnalyzeBugs(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*BugAnalysisResult, error)
}

// ErrorAnalyzer analyzes session entries for errors.
type ErrorAnalyzer interface {
	AnalyzeErrors(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*ErrorAnalysisResult, error)
}

// QualityScanner performs quality scan on session entries.
type QualityScanner interface {
	QualityScan(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*QualityScanResult, error)
}

// WorkPatternsAnalyzer analyzes work patterns from session entries.
type WorkPatternsAnalyzer interface {
	GetWorkPatterns(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*WorkPatternsResult, error)
}

// TimelineAnalyzer builds a timeline from session entries.
type TimelineAnalyzer interface {
	GetTimeline(entries []parser.SessionEntry, limit int) (*TimelineResult, error)
}

// TechDebtAnalyzer analyzes technical debt in session entries.
type TechDebtAnalyzer interface {
	GetTechDebt(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*TechDebtResult, error)
}

// DefaultAnalyzer implements all analyzer interfaces by delegating to package-level functions.
type DefaultAnalyzer struct{}

// Compile-time assertions that DefaultAnalyzer implements all interfaces.
var _ BugAnalyzer = (*DefaultAnalyzer)(nil)
var _ ErrorAnalyzer = (*DefaultAnalyzer)(nil)
var _ QualityScanner = (*DefaultAnalyzer)(nil)
var _ WorkPatternsAnalyzer = (*DefaultAnalyzer)(nil)
var _ TimelineAnalyzer = (*DefaultAnalyzer)(nil)
var _ TechDebtAnalyzer = (*DefaultAnalyzer)(nil)

// AnalyzeBugs delegates to the package-level AnalyzeBugs function.
func (d *DefaultAnalyzer) AnalyzeBugs(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*BugAnalysisResult, error) {
	return AnalyzeBugs(entries, toolCalls, limit)
}

// AnalyzeErrors delegates to the package-level AnalyzeErrors function.
func (d *DefaultAnalyzer) AnalyzeErrors(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*ErrorAnalysisResult, error) {
	return AnalyzeErrors(entries, toolCalls, limit)
}

// QualityScan delegates to the package-level QualityScan function.
func (d *DefaultAnalyzer) QualityScan(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*QualityScanResult, error) {
	return QualityScan(entries, toolCalls)
}

// GetWorkPatterns delegates to the package-level GetWorkPatterns function.
func (d *DefaultAnalyzer) GetWorkPatterns(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*WorkPatternsResult, error) {
	return GetWorkPatterns(entries, toolCalls)
}

// GetTimeline delegates to the package-level GetTimeline function.
func (d *DefaultAnalyzer) GetTimeline(entries []parser.SessionEntry, limit int) (*TimelineResult, error) {
	return GetTimeline(entries, limit)
}

// GetTechDebt delegates to the package-level GetTechDebt function.
func (d *DefaultAnalyzer) GetTechDebt(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*TechDebtResult, error) {
	return GetTechDebt(entries, toolCalls)
}

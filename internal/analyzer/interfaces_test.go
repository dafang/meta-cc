package analyzer_test

import "github.com/yaleh/meta-cc/internal/analyzer"

// Compile-time assertions that DefaultAnalyzer implements all analyzer interfaces.
var _ analyzer.BugAnalyzer = (*analyzer.DefaultAnalyzer)(nil)
var _ analyzer.ErrorAnalyzer = (*analyzer.DefaultAnalyzer)(nil)
var _ analyzer.QualityScanner = (*analyzer.DefaultAnalyzer)(nil)
var _ analyzer.WorkPatternsAnalyzer = (*analyzer.DefaultAnalyzer)(nil)
var _ analyzer.TimelineAnalyzer = (*analyzer.DefaultAnalyzer)(nil)
var _ analyzer.TechDebtAnalyzer = (*analyzer.DefaultAnalyzer)(nil)

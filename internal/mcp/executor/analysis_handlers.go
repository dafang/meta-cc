package executor

import "context"

func init() {
	registerHandler("analyze_bugs", handleAnalyzeBugs)
	registerHandler("analyze_errors", handleAnalyzeErrors)
	registerHandler("quality_scan", handleQualityScan)
	registerHandler("get_work_patterns", handleGetWorkPatterns)
	registerHandler("get_timeline", handleGetTimeline)
	registerHandler("get_tech_debt", handleGetTechDebt)
}

func handleAnalyzeBugs(_ context.Context, e *ToolExecutor, params map[string]interface{}) (string, error) {
	return e.AnalysisSvc.AnalyzeBugs(params)
}

func handleAnalyzeErrors(_ context.Context, e *ToolExecutor, params map[string]interface{}) (string, error) {
	return e.AnalysisSvc.AnalyzeErrors(params)
}

func handleQualityScan(_ context.Context, e *ToolExecutor, params map[string]interface{}) (string, error) {
	return e.AnalysisSvc.QualityScan(params)
}

func handleGetWorkPatterns(_ context.Context, e *ToolExecutor, params map[string]interface{}) (string, error) {
	return e.AnalysisSvc.GetWorkPatterns(params)
}

func handleGetTimeline(_ context.Context, e *ToolExecutor, params map[string]interface{}) (string, error) {
	return e.AnalysisSvc.GetTimeline(params)
}

func handleGetTechDebt(_ context.Context, e *ToolExecutor, params map[string]interface{}) (string, error) {
	return e.AnalysisSvc.GetTechDebt(params)
}

# Automation Analysis - Iteration 4

**Date**: 2025-10-18
**Experiment**: Bootstrap-002 Test Strategy Development
**Focus**: Meta Layer Convergence through Tool Automation

---

## Pattern Review (8 Patterns Documented)

### From Iteration 1 (Patterns 1-5)
1. **Unit Test Pattern**: Simple function testing
2. **Table-Driven Test Pattern**: Multiple scenarios
3. **Integration Test Pattern**: Complete flows (MCP server)
4. **Error Path Test Pattern**: Systematic error coverage
5. **Test Helper Pattern**: Reduce duplication

### From Iteration 2 (Pattern 6)
6. **Dependency Injection Pattern**: Mock external dependencies

### From Iteration 3 (Patterns 7-8)
7. **CLI Command Test Pattern**: In-process CLI testing
8. **Global Flag Test Pattern**: Flag parsing and propagation

---

## Automation Opportunities Identified

### 1. Test Generator Script
**Purpose**: Generate test scaffold from function signature
**Input**: Go function signature or file path
**Output**: Test file with appropriate pattern applied

**Example**:
```bash
./scripts/generate-test.sh "func ParseQuery(q string) (Query, error)" --pattern=table-driven
```

**Generated Output**:
```go
func TestParseQuery(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected Query
        wantErr  bool
    }{
        {
            name:     "valid query",
            input:    "SELECT * FROM users",
            expected: Query{/* ... */},
            wantErr:  false,
        },
        {
            name:     "invalid query",
            input:    "",
            expected: Query{},
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseQuery(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("ParseQuery() = %v, expected %v", result, tt.expected)
            }
        })
    }
}
```

**Supports**:
- Pattern 1: Unit test
- Pattern 2: Table-driven
- Pattern 4: Error path
- Pattern 7: CLI command
- Pattern 8: Global flag

### 2. Coverage Gap Analyzer
**Purpose**: Identify functions with <80% coverage and prioritize
**Input**: Coverage report (coverage.out)
**Output**: Prioritized list of functions to test

**Example**:
```bash
./scripts/analyze-coverage-gaps.sh coverage.out
```

**Output**:
```
COVERAGE GAP ANALYSIS - 2025-10-18

Total Coverage: 72.5%
Target: 80.0%
Gap: -7.5 percentage points

HIGH PRIORITY (Error Handling - 0-60% coverage):
1. internal/validation/validate.go:ValidateInput (0.0%) - P1 Error Handling
2. cmd/mcp-server/executor.go:handleError (25.0%) - P1 Error Handling
3. cmd/mcp-server/capabilities.go:validateCapability (40.0%) - P1 Error Handling

MEDIUM PRIORITY (Business Logic - 0-75% coverage):
4. internal/query/parser.go:ParseQuery (55.0%) - P2 Business Logic
5. cmd/stats.go:calculateStats (60.0%) - P2 Business Logic

LOW PRIORITY (Utilities - 0-60% coverage):
6. internal/util/format.go:FormatOutput (45.0%) - P3 Utility

RECOMMENDED TEST PATTERNS:
- ValidateInput: Error Path Pattern (Pattern 4) + Table-Driven (Pattern 2)
- handleError: Error Path Pattern (Pattern 4)
- ParseQuery: Table-Driven Pattern (Pattern 2)

ESTIMATED WORK:
- High priority: 3 functions × 15 min = 45 min → +3.5% coverage
- Medium priority: 2 functions × 12 min = 24 min → +2.0% coverage
- Total estimated: 69 min → +5.5% coverage (reach 78.0%)
```

**Features**:
- Parses coverage.out using go tool cover
- Categorizes by priority (P1-P4)
- Estimates coverage impact
- Suggests appropriate patterns
- Calculates time estimates

### 3. Test Template Tool
**Purpose**: Interactive template selection and generation
**Input**: User responses to prompts
**Output**: Test file with selected pattern

**Example**:
```bash
./scripts/test-template-tool.sh
```

**Interactive Flow**:
```
Test Template Generator
=======================

What are you testing?
1) Unit function
2) CLI command
3) Error paths
4) Integration flow
5) Global flags

Choice: 3

Error Path Test Pattern Selected
---------------------------------

Function name: ValidateInput
Package: internal/validation
Number of error scenarios (2-6): 4

Generating test file...

Created: internal/validation/validate_test.go (120 lines)

Test includes:
- 4 error scenarios (nil input, empty input, invalid format, out of range)
- Table-driven structure
- Error message validation
- wantErr patterns

Next steps:
1. Fill in test cases with actual values
2. Run: go test ./internal/validation/
3. Verify coverage: go test -cover ./internal/validation/
```

**Supports**:
- All 8 patterns
- Interactive mode
- Non-interactive mode (flags)
- Template customization

---

## Tool Design Specifications

### Tool 1: generate-test.sh

**Interface**:
```bash
Usage: generate-test.sh [OPTIONS] FUNCTION_SIG

Options:
  --pattern PATTERN    Test pattern (unit, table-driven, error-path, cli-command, global-flag)
  --package PACKAGE    Package name (default: infer from current dir)
  --output FILE        Output file (default: <package>_test.go)
  --append             Append to existing file instead of creating new
  --dry-run            Print to stdout instead of writing file

Examples:
  generate-test.sh "func Parse(s string) (int, error)" --pattern=table-driven
  generate-test.sh --file cmd/stats.go --function calculateStats --pattern=unit
```

**Implementation**:
- Parse function signature
- Extract: function name, parameters, return types
- Select template based on pattern
- Generate test with placeholders
- Format with gofmt

**Templates**:
- unit-test.tmpl
- table-driven.tmpl
- error-path.tmpl
- cli-command.tmpl
- global-flag.tmpl

### Tool 2: analyze-coverage-gaps.sh

**Interface**:
```bash
Usage: analyze-coverage-gaps.sh [OPTIONS] COVERAGE_FILE

Options:
  --threshold PCT      Coverage threshold (default: 80)
  --top N              Show top N functions (default: 10)
  --category CAT       Filter by category (error-handling, business-logic, cli, etc.)
  --json               Output as JSON
  --estimate           Show time/coverage estimates

Examples:
  analyze-coverage-gaps.sh coverage.out
  analyze-coverage-gaps.sh coverage.out --threshold 70 --top 5
  analyze-coverage-gaps.sh coverage.out --category error-handling --json
```

**Implementation**:
- Parse coverage.out with `go tool cover -func`
- Categorize functions by:
  - File path (cmd/, internal/validation, etc.)
  - Function name patterns (Validate*, Handle*, Parse*, etc.)
- Assign priority (P1-P4) based on category
- Calculate coverage gap per function
- Estimate time and coverage impact
- Suggest patterns based on function type

**Categorization Rules**:
```go
// P1: Error Handling (80-90% target)
if strings.HasPrefix(funcName, "Validate") ||
   strings.HasPrefix(funcName, "Handle") ||
   strings.Contains(file, "validation") {
    return "error-handling", 1
}

// P2: Business Logic (75-85% target)
if strings.Contains(file, "query") ||
   strings.Contains(file, "analyzer") ||
   strings.HasPrefix(funcName, "Process") {
    return "business-logic", 2
}

// P3: CLI (70-80% target)
if strings.Contains(file, "cmd/") && !strings.Contains(file, "cmd/mcp-server") {
    return "cli", 2
}

// P4: Infrastructure (best effort)
if strings.HasPrefix(funcName, "Init") ||
   strings.Contains(funcName, "Logger") {
    return "infrastructure", 4
}
```

### Tool 3: test-template-tool.sh

**Interface**:
```bash
Usage: test-template-tool.sh [OPTIONS]

Options:
  --interactive        Interactive mode (default)
  --pattern PATTERN    Pattern to use (skip interactive)
  --function NAME      Function name
  --package PACKAGE    Package path
  --scenarios N        Number of test scenarios (for table-driven)

Examples:
  test-template-tool.sh                                    # Interactive
  test-template-tool.sh --pattern error-path --function ValidateInput --scenarios 4
```

**Implementation**:
- Interactive prompts using `select` (bash) or Go CLI
- Template rendering with variable substitution
- File creation with gofmt
- Summary of generated test

---

## Success Metrics

### Tool Usage Metrics (to be measured)

**Test Generator**:
- Time to generate test: <5 seconds
- Generated test compiles: 100%
- Manual edits needed: <20% of lines
- Time saved vs manual: 5-10 minutes per test

**Coverage Gap Analyzer**:
- Analysis time: <2 seconds for typical project
- Accuracy of categorization: >90%
- Estimate accuracy: ±20% time, ±15% coverage
- Actionability: Clear next steps provided

**Test Template Tool**:
- Interactive flow time: <2 minutes
- Template quality: Compiles without errors
- Pattern application: Correct pattern for scenario
- User satisfaction: Reduces decision time by 50%

### Effectiveness Measurement Plan

**Measure in Iteration 4**:
1. Generate 5 tests manually (baseline)
2. Generate 5 tests with tools
3. Compare:
   - Time per test (manual vs automated)
   - Lines of code written
   - Compilation errors
   - Coverage achieved
   - Pattern compliance

**Target Speedup**: 5x (from 15 min/test to 3 min/test)

---

## Implementation Priority

**Phase 1** (Core Functionality):
1. Coverage Gap Analyzer - Highest ROI, minimal dependencies
2. Test Generator - High value, enables automation

**Phase 2** (Enhanced UX):
3. Test Template Tool - Interactive, improves adoption

**Time Estimate**:
- Phase 1: ~3 hours (analyzer 1.5h, generator 1.5h)
- Phase 2: ~1.5 hours
- Total: ~4.5 hours

---

## Reusability Assessment

### Cross-Project Applicability

**Coverage Gap Analyzer**:
- Language: Go-specific (uses `go tool cover`)
- Framework: Framework-agnostic
- Transferability: 100% to other Go projects
- Adaptation: Minimal (category rules may need tuning)

**Test Generator**:
- Language: Go-specific (generates Go test code)
- Framework: Testing framework agnostic
- Transferability: 100% to other Go projects
- Adaptation: Templates may need customization for project style

**Test Template Tool**:
- Language: Go-specific
- Framework: Generic test patterns
- Transferability: 90% to other Go projects, 60% to other languages (concept)
- Adaptation: Templates need language-specific syntax

### Cross-Language Adaptation

**Concept Reusability**: 100%
- Coverage gap analysis: Universal concept
- Pattern-based generation: Applies to any language
- Prioritization framework: Language-agnostic

**Implementation Reusability**: 40-60%
- Tool structure: Reusable
- Parsing logic: Language-specific
- Templates: Language-specific
- Categorization: Adapt to language idioms

**Estimated Adaptation Effort**:
- Go → Python: ~40% rewrite (different coverage tools, test syntax)
- Go → JavaScript: ~50% rewrite (jest/mocha patterns)
- Go → Rust: ~35% rewrite (similar patterns, different syntax)

---

## Next Steps

1. ✅ Document automation opportunities
2. ⏳ Implement Coverage Gap Analyzer
3. ⏳ Implement Test Generator
4. ⏳ Implement Test Template Tool
5. ⏳ Measure effectiveness (manual vs automated)
6. ⏳ Validate reusability in different package
7. ⏳ Calculate V_meta(s₄)

**Expected V_meta Components**:
- V_completeness: 0.70 → 0.80 (3 tools created)
- V_effectiveness: 0.40 → 0.60 (5x speedup measured)
- V_reusability: 0.40 → 0.60 (validated in 2+ contexts)

---

**Status**: Analysis Complete - Ready for Implementation

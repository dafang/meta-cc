# Baseline Metrics - Iteration 0

**Package**: `internal/query/`
**Date**: 2025-10-19
**Total Lines**: 1,810 (including tests)

## Summary

| Metric | Value |
|--------|-------|
| Total Files | 7 |
| Total Functions | 47 (including test functions) |
| Average Cyclomatic Complexity | 4.8 |
| Test Coverage | 92.0% |
| Code Duplication | 31 clone groups |
| Static Analysis Warnings (go vet) | 0 |

## Cyclomatic Complexity Analysis

### Statistics
- **Total functions analyzed**: 47
- **Average complexity**: 4.8
- **Functions with complexity >10**: 5 (10.6%)
- **Functions with complexity >15**: 0
- **Maximum complexity**: 13 (TestBuildToolSequenceQuery)

### High Complexity Functions (>10)

| Complexity | Function | File | Type |
|------------|----------|------|------|
| 13 | TestBuildToolSequenceQuery | sequences_test.go | Test |
| 12 | TestBuildContextQuery | context_test.go | Test |
| 11 | TestBuildToolSequenceQueryEmptyPatternExcludesBuiltin | sequences_test.go | Test |
| 11 | TestBuildFileAccessQuery | file_access_test.go | Test |
| 10 | calculateSequenceTimeSpan | sequences.go | Production |

### Analysis
- 4 out of 5 high-complexity functions are test functions
- Only 1 production function has complexity >10: `calculateSequenceTimeSpan` (10)
- Production code average complexity is likely lower than overall average (test functions skew the data)

## Code Duplication

**Total clone groups**: 31

### Duplication by File Type
- Test files: Majority of duplication (test setup code, table-driven test patterns)
- Production files: 6 clone groups

### Notable Duplication Patterns

1. **Test Setup Boilerplate** (multiple occurrences)
   - File: context_test.go, file_access_test.go, sequences_test.go
   - Lines: 3-7 lines repeated test case setups

2. **Error Checking Pattern** (4 clones)
   - Files: context_test.go, file_access_test.go, sequences_test.go
   - Lines: 239-241, 216-218, 322-324, 198-200
   - Pattern: `if err != nil { t.Fatalf(...) }`

3. **Production Code Duplication** (3 clones)
   - Files: file_access.go, sequences.go
   - Lines: 61-63, 46-48, 171-173
   - Pattern: Similar error handling or data extraction logic

### Impact
- Most duplication is in test files (acceptable for test clarity)
- Some production code duplication should be addressed (6 groups)

## Static Analysis

### Staticcheck
- **Status**: Tool incompatibility (requires go1.24.0, using go1.23.1)
- **Result**: Cannot run

### Go Vet
- **Warnings**: 0
- **Status**: Clean

### Interpretation
- No vet warnings indicates good code quality baseline
- Missing staticcheck data is a gap in analysis

## Test Coverage

### Overall Coverage
- **Coverage**: 92.0% of statements
- **Status**: Excellent baseline

### Per-File Coverage

| File | Coverage |
|------|----------|
| context.go | High (all functions >70%) |
| file_access.go | High (all functions >75%) |
| sequences.go | High (most functions 100%) |

### Functions with Lower Coverage (<90%)

| Function | Coverage | File |
|----------|----------|------|
| buildTurnPreview | 72.7% | context.go |
| parseTimestamp | 75.0% | context.go |
| lastSlash | 75.0% | file_access.go |
| getToolCallTimestamp | 75.0% | file_access.go |
| findTimestampForTurn | 75.0% | sequences.go |
| findErrorOccurrences | 85.7% | context.go |
| calculateSequenceTimeSpan | 85.0% | sequences.go |

### Analysis
- 7 functions have coverage <90%
- All are >70%, no critical gaps
- Most uncovered lines likely edge cases or error paths

## File Statistics

| File | Lines | Type |
|------|-------|------|
| sequences_test.go | 553 | Test |
| file_access_test.go | 327 | Test |
| sequences.go | 259 | Production |
| context_test.go | 244 | Test |
| context.go | 215 | Production |
| file_access.go | 155 | Production |
| types.go | 57 | Production |
| **Total** | **1,810** | |

### Production vs Test Code
- **Production code**: 686 lines (37.9%)
- **Test code**: 1,124 lines (62.1%)
- **Test-to-code ratio**: 1.64:1 (good coverage commitment)

### File Size Distribution
- **Large files (>300 lines)**: 2 (sequences_test.go: 553, file_access_test.go: 327)
- **Medium files (200-300 lines)**: 3 (sequences.go: 259, context_test.go: 244, context.go: 215)
- **Small files (<200 lines)**: 2 (file_access.go: 155, types.go: 57)

## Baseline Assessment

### Strengths
✓ High test coverage (92%)
✓ Low average complexity (4.8)
✓ No go vet warnings
✓ Good test-to-code ratio
✓ Only 1 production function with complexity >10

### Weaknesses
✗ 31 duplication clone groups (6 in production code)
✗ 1 high-complexity production function (calculateSequenceTimeSpan: 10)
✗ 7 functions with coverage <90%
✗ Cannot run staticcheck (tool version incompatibility)
✗ Test files are quite large (sequences_test.go: 553 lines)

### Opportunities for Refactoring
1. **Primary Target**: `calculateSequenceTimeSpan` (complexity 10, coverage 85%)
2. **Secondary Target**: Production code duplication (6 clone groups)
3. **Tertiary Target**: Improve coverage for 7 functions <90%
4. **Test Organization**: Consider splitting large test files (sequences_test.go: 553 lines)

## Metrics for Value Function Calculation

### V_code_quality Components
- **Baseline complexity**: 4.8 average, 1 function >10
- **Baseline duplication**: 31 clone groups (6 in production)
- **Baseline static warnings**: 0 (go vet)

### V_maintainability Components
- **Baseline coverage**: 92.0%
- **Module cohesion**: Appears good (3 separate files with clear responsibilities)
- **Documentation**: To be assessed manually

### V_safety Components
- **Test pass rate**: 100% (all tests passing)
- **Behavior preservation**: Baseline established
- **Git discipline**: To be measured during refactoring

### V_effort Components
- **Baseline time estimate**: To be measured during initial refactoring attempt
- **Automation**: None yet (ad-hoc approach)
- **Rework**: To be measured

## Data Sources

All metrics collected on 2025-10-19:
- `complexity-baseline.txt`: gocyclo output
- `duplication-baseline.txt`: dupl output
- `govet-baseline.txt`: go vet output
- `coverage-baseline.txt`: go test coverage output
- `coverage.out`: detailed coverage profile
- `file-stats.txt`: line counts

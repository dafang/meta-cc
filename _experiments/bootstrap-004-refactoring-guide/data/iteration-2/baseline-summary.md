# Baseline Metrics - Iteration 2

**Date**: 2025-10-19
**Target Function**: `calculateSequenceTimeSpan` in `internal/query/sequences.go`
**Refactoring Pattern**: Extract Method

---

## Pre-Refactoring State

### Test Status
- **All tests passing**: ✅ PASS
- **Test count**: 19 test functions
- **Test duration**: ~0.006s (cached)
- **Exit code**: 0

### Coverage Metrics
- **Overall coverage**: 92.0%
- **Target function**: `calculateSequenceTimeSpan` - 85.0%
- **Related function**: `findTimestampForTurn` - 75.0%

**Gap Analysis**:
- Target coverage for refactoring: ≥95%
- Current: 85%
- **GAP**: Need +10% coverage before refactoring

### Complexity Metrics
- **Average complexity**: 4.8
- **Target function**: `calculateSequenceTimeSpan` - **10** (HIGHEST production complexity)
- **File**: `internal/query/sequences.go:221:1`

**High Complexity Functions** (production only):
1. calculateSequenceTimeSpan: **10** ← TARGET FOR REFACTORING

**Test Functions** (excluded from refactoring):
- TestBuildToolSequenceQuery: 13
- TestBuildContextQuery: 12
- TestBuildToolSequenceQueryEmptyPatternExcludesBuiltin: 11
- TestBuildFileAccessQuery: 11

### Git Status
- **Branch**: `refactor/bootstrap-004-iteration-2-calculateSequenceTimeSpan`
- **Status**: Clean (iteration 1 artifacts committed)
- **Uncommitted changes**: None
- **Safe to proceed**: ✅ YES

---

## Refactoring Plan

### Target Complexity Reduction
- **Current**: 10
- **Target**: <8 (ideally ≤6)
- **Expected reduction**: 40%+

### Extract Method Opportunities

**Function**: `calculateSequenceTimeSpan` (lines 221-259, 39 lines)

**Identified Responsibilities**:
1. **Collect timestamps** (lines 227-240): Loop through occurrences, get start/end timestamps
2. **Find min/max** (lines 247-256): Find minimum and maximum from timestamp slice
3. **Calculate span** (line 258): Convert difference to minutes

**Proposed Extractions**:
1. Extract `collectOccurrenceTimestamps(occurrences, entries, toolCalls) []int64`
   - Lines 227-240
   - Expected complexity: 4-5
   - Reduces main function to ~6

2. Extract `findMinMaxTimestamps(timestamps []int64) (int64, int64)`
   - Lines 247-256
   - Expected complexity: 2-3
   - Further reduces main function to ~4

### Coverage Improvement Plan

**Phase 1b** (TDD Workflow): Write Missing Tests FIRST
- **Edge case 1**: Empty occurrences (should return 0)
- **Edge case 2**: Single occurrence (start == end, should return 0)
- **Edge case 3**: Occurrences with no valid timestamps (all return 0)
- **Edge case 4**: Large time span (verify minutes calculation correct)

**Target**: 85% → 95% (+10%)

---

## Safety Checklist Status

### Pre-Refactoring Checklist

- [x] **All tests passing**: PASS
- [x] **No uncommitted changes**: CLEAN
- [x] **Baseline metrics recorded**: Saved to `data/iteration-2/`
  - tests-baseline.txt
  - coverage-baseline.txt
  - complexity-baseline.txt
  - complexity-avg-baseline.txt
- [x] **Target code has tests**: YES (`sequences_test.go`)
- [x] **Tests cover current behavior**: 85% (need to improve to 95%)
- [x] **Refactoring pattern selected**: Extract Method
- [x] **Incremental steps defined**:
  1. Write edge case tests (Phase 1b)
  2. Extract collectOccurrenceTimestamps
  3. Write tests for extracted function
  4. Extract findMinMaxTimestamps
  5. Write tests for extracted function
  6. Simplify main function
  7. Verify final complexity <8
- [x] **Rollback plan documented**: Git revert on each commit if tests fail

### During Refactoring
- Will follow per-step checklist
- Each step <10 minutes
- Test after each change
- Commit after passing tests

### Post-Refactoring
- Final verification checklist
- Metrics comparison
- Documentation update

---

## Expected Outcomes

### Complexity
- **calculateSequenceTimeSpan**: 10 → 4-6 (40-60% reduction)
- **New function 1**: collectOccurrenceTimestamps: 4-5
- **New function 2**: findMinMaxTimestamps: 2-3

### Coverage
- **Overall**: 92.0% → 93-94% (edge case tests)
- **Target function**: 85% → 95%+ (edge cases covered)

### Test Count
- **Current**: 19 test functions
- **Expected**: 21-23 test functions (+2-4 characterization tests)

### Commits
- **Expected**: 6-8 small commits
- **Average size**: 20-50 lines per commit
- **All commits**: Passing tests

---

## Automation

**Complexity Checking**:
```bash
/home/yale/go/bin/gocyclo -over 10 internal/query/sequences.go
```

**Coverage Checking**:
```bash
go test -cover ./internal/query/...
go test -coverprofile=/tmp/coverage.out ./internal/query && go tool cover -func=/tmp/coverage.out | grep calculateSequenceTimeSpan
```

**Full Test Suite**:
```bash
go test -v ./internal/query/...
```

---

**Status**: READY TO PROCEED with Phase 1b (Write Missing Tests)
**Next Step**: Add edge case tests to improve coverage from 85% to 95%

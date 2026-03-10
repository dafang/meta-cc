# Code Smells Identified - Iteration 0

**Package**: `internal/query/`
**Date**: 2025-10-19
**Analysis Method**: Manual inspection + automated metrics

## Summary

| Category | Count | Priority |
|----------|-------|----------|
| High Complexity Functions | 1 | High |
| Code Duplication (Production) | 6 groups | Medium-High |
| Long Functions | 1 | Medium |
| Poor Naming | 3 instances | Low |
| Missing Edge Case Coverage | 7 functions | Medium |
| God Objects | 0 | N/A |
| Primitive Obsession | 2 instances | Low |

## HIGH PRIORITY SMELLS

### 1. High Cyclomatic Complexity

#### calculateSequenceTimeSpan (sequences.go:221-259)
- **Complexity**: 10
- **Coverage**: 85%
- **Lines**: 39
- **Issue**: Multiple nested loops and conditionals
- **Impact**: Hard to test, hard to understand, maintenance burden

**Problematic Pattern**:
```go
func calculateSequenceTimeSpan(...) int {
    if len(occurrences) == 0 {
        return 0
    }

    // Collect all relevant timestamps
    var timestamps []int64

    for _, occ := range occurrences {              // Loop 1
        startTs := findTimestampForTurn(...)
        endTs := findTimestampForTurn(...)

        if startTs > 0 {                            // Conditional 1
            timestamps = append(timestamps, startTs)
        }
        if endTs > 0 && endTs != startTs {         // Conditional 2 (compound)
            timestamps = append(timestamps, endTs)
        }
    }

    if len(timestamps) == 0 {                       // Conditional 3
        return 0
    }

    // Find min and max
    minTs := timestamps[0]
    maxTs := timestamps[0]
    for _, ts := range timestamps[1:] {             // Loop 2
        if ts < minTs {                             // Conditional 4
            minTs = ts
        }
        if ts > maxTs {                             // Conditional 5
            maxTs = ts
        }
    }

    return int((maxTs - minTs) / SecondsPerMinute)
}
```

**Refactoring Opportunities**:
1. Extract method: `collectTimestamps(occurrences, entries, toolCalls) []int64`
2. Extract method: `findMinMax(timestamps []int64) (min, max int64)`
3. Simplify edge case handling
4. Consider using Go's built-in `min()` and `max()` functions (Go 1.21+)

**Expected Impact**:
- Reduce complexity from 10 to ~4-5
- Improve testability (can test min/max logic separately)
- Improve coverage from 85% to 95%+

---

## MEDIUM-HIGH PRIORITY SMELLS

### 2. Code Duplication - Error Handling Pattern

#### Duplicate: Error Return Pattern (3 occurrences)
**Files**:
- `file_access.go:61-63`
- `sequences.go:46-48`
- `sequences.go:171-173`

**Pattern**:
```go
sort.Slice(result, func(i, j int) bool {
    return result[i].Count > result[j].Count
})
```

**Issue**: Same sorting logic duplicated for sequences
**Impact**: If sorting criteria changes, must update multiple places

**Refactoring Opportunity**:
- Extract to `sortSequencesByCount(sequences []types.SequencePattern)`
- Encapsulate sorting behavior

---

### 3. Code Duplication - buildContextBefore/After Pattern

#### Duplicate: Context Building (2 occurrences)
**Files**:
- `context.go:126-128` (buildContextBefore)
- `context.go:131-133` (buildContextAfter)

**Pattern**:
```go
func buildContextBefore(...) []TurnPreview {
    return buildContextWindow(entries, errorTurn, window, turnIndex, "before")
}

func buildContextAfter(...) []TurnPreview {
    return buildContextWindow(entries, errorTurn, window, turnIndex, "after")
}
```

**Issue**: Two wrapper functions doing almost identical work
**Impact**: Adds indirection without significant value

**Refactoring Opportunity**:
- Consider removing wrappers entirely
- Call `buildContextWindow` directly with direction parameter
- Or use constants for "before"/"after" instead of string literals

---

### 4. Code Duplication - Turn Index Building

#### Duplicate: buildTurnIndex called in multiple places
**Files**:
- `context.go:34` (BuildContextQuery)
- `sequences.go:40` (BuildToolSequenceQuery)
- `file_access.go:19` (BuildFileAccessQuery)

**Pattern**:
```go
turnIndex := buildTurnIndex(entries)
```

**Issue**: Each query function rebuilds turn index
**Impact**: Performance - O(n) work repeated per query

**Refactoring Opportunity**:
- Cache turn index at a higher level
- Pass pre-built index as parameter
- Or make buildTurnIndex more efficient (lazy evaluation)

**Note**: This may be acceptable if queries are independent, but worth considering for batch operations.

---

## MEDIUM PRIORITY SMELLS

### 5. Long Function

#### buildTurnPreview (context.go:136-167)
- **Lines**: 32
- **Coverage**: 72.7% (lowest among production functions)
- **Complexity**: 8
- **Issue**: Does too many things (extraction + transformation)

**Responsibilities**:
1. Initialize preview struct
2. Extract role
3. Extract preview text from blocks
4. Extract tools from blocks
5. Parse timestamp

**Refactoring Opportunity**:
- Extract method: `extractPreviewFromBlocks(blocks []ContentBlock) (string, []string)`
- Simplify conditional logic
- Improve testability of preview extraction

**Expected Impact**:
- Improve coverage from 72.7% to 90%+
- Reduce complexity from 8 to ~5
- Make preview extraction logic reusable

---

### 6. Functions with Lower Coverage (<90%)

These functions have adequate coverage but could be improved:

| Function | Coverage | File | Potential Issue |
|----------|----------|------|-----------------|
| buildTurnPreview | 72.7% | context.go | Complex branching logic |
| parseTimestamp | 75.0% | context.go | Error handling not fully tested |
| lastSlash | 75.0% | file_access.go | Edge cases (empty string, no slash) |
| getToolCallTimestamp | 75.0% | file_access.go | Not found case |
| findTimestampForTurn | 75.0% | sequences.go | Not found case |
| calculateSequenceTimeSpan | 85.0% | sequences.go | Complex logic, edge cases |
| findErrorOccurrences | 85.7% | context.go | Error filtering logic |

**Refactoring Opportunity**:
- Add tests for edge cases
- Simplify conditional logic to improve testability
- Extract complex conditionals to named functions

---

## LOW PRIORITY SMELLS

### 7. Poor Naming - Magic Strings

#### Direction Parameter in buildContextWindow
**File**: `context.go:97`

**Issue**:
```go
func buildContextWindow(..., direction string) []TurnPreview {
    // ...
    if direction == "before" {
        // ...
    } else { // "after"
        // ...
    }
}
```

**Problem**: String literals "before" and "after" are magic values
**Impact**: Typos not caught at compile time, harder to understand

**Refactoring Opportunity**:
```go
type Direction int

const (
    DirectionBefore Direction = iota
    DirectionAfter
)

func buildContextWindow(..., direction Direction) []TurnPreview {
    if direction == DirectionBefore {
        // ...
    } else {
        // ...
    }
}
```

---

### 8. Primitive Obsession - Turn Representation

**Files**: Throughout package

**Issue**: Turn numbers represented as `int` everywhere
**Impact**: Type safety - any int could be a turn number

**Example**:
```go
func buildContextBefore(entries []parser.SessionEntry, errorTurn, window int, ...) []TurnPreview
```

**Refactoring Opportunity** (Low priority):
```go
type Turn int

func buildContextBefore(entries []parser.SessionEntry, errorTurn Turn, window int, ...) []TurnPreview
```

**Note**: This is a minor improvement and may not provide significant value given current codebase size.

---

### 9. Primitive Obsession - UUID Representation

**Issue**: UUIDs represented as `string` everywhere
**Impact**: Type safety - any string could be mistaken for UUID

**Refactoring Opportunity** (Very low priority):
```go
type UUID string

func buildTurnIndex(entries []parser.SessionEntry) map[UUID]int
```

---

## POSITIVE OBSERVATIONS (Not Smells)

### Strengths of Current Code

1. **Good Separation of Concerns**:
   - `context.go`: Context query building
   - `sequences.go`: Sequence pattern finding
   - `file_access.go`: File access tracking
   - Clear single responsibility per file

2. **Consistent Naming**:
   - Build* for query constructors
   - build* (lowercase) for internal helpers
   - extract* for data extraction
   - Good verb-noun patterns

3. **Good Use of Helper Functions**:
   - Small, focused helpers (parsePattern, matchesSequence, etc.)
   - Generally < 20 lines each

4. **Proper Error Handling**:
   - Uses custom error wrapping (mcerrors)
   - Validates inputs in public functions

5. **Good Test Coverage**:
   - 92% overall
   - Most critical paths covered

## PRIORITIZED REFACTORING TARGETS

### Iteration 0 Focus

**Target 1: calculateSequenceTimeSpan** (HIGH PRIORITY)
- Complexity: 10 → 4-5
- Coverage: 85% → 95%+
- Effort: ~30-60 minutes
- Impact: High (reduces highest complexity, improves testability)

**Target 2: Production Code Duplication** (MEDIUM-HIGH PRIORITY)
- Address 6 duplication groups
- Effort: ~20-40 minutes
- Impact: Medium (reduces maintenance burden)

### Future Iterations

**Target 3: buildTurnPreview** (MEDIUM PRIORITY)
- Coverage: 72.7% → 90%+
- Complexity: 8 → 5
- Effort: ~30-45 minutes

**Target 4: Improve edge case coverage** (MEDIUM PRIORITY)
- Add tests for 7 functions with <90% coverage
- Effort: ~60-90 minutes

**Target 5: Remove magic strings** (LOW PRIORITY)
- Replace string direction with enum/const
- Effort: ~15-20 minutes

## SMELL CATEGORIES BREAKDOWN

### By Martin Fowler's Catalog

| Refactoring Category | Instances | Examples |
|---------------------|-----------|----------|
| Long Method | 1 | calculateSequenceTimeSpan (39 lines, complexity 10) |
| Duplicated Code | 6 groups | Sort logic, context builders, turn index |
| Primitive Obsession | 2 | Turn as int, UUID as string |
| Long Parameter List | 0 | None identified |
| Large Class | 0 | Files are appropriately sized |
| Data Clumps | 0 | None identified |
| Feature Envy | 0 | None identified |

### By Impact

| Impact Level | Count | Action |
|--------------|-------|--------|
| High | 1 | Address in Iteration 0 |
| Medium-High | 3 | Address in Iteration 0-1 |
| Medium | 2 | Address in Iteration 1-2 |
| Low | 3 | Address in later iterations or defer |

## REFACTORING STRATEGY NOTES

### Safe Refactoring Prerequisites
1. ✓ Existing tests (92% coverage)
2. ✓ Tests passing (100% pass rate)
3. ✓ Clear functionality boundaries
4. ✓ No external dependencies in refactoring targets

### Risk Assessment
- **Low Risk**: calculateSequenceTimeSpan (well-tested, isolated)
- **Low Risk**: Duplication removal (covered by tests)
- **Low Risk**: buildTurnPreview (covered by tests)
- **Medium Risk**: Turn index caching (performance optimization, needs benchmarks)

### Incremental Approach
1. Start with highest complexity function (calculateSequenceTimeSpan)
2. Write additional tests for edge cases
3. Refactor with TDD cycle
4. Verify no regression
5. Commit incrementally

## CONCLUSION

The `internal/query/` package is in good shape overall:
- Strong test coverage (92%)
- Low average complexity (4.8)
- Clean separation of concerns
- Only 1 high-complexity production function

Primary refactoring target: **calculateSequenceTimeSpan** (complexity 10, coverage 85%)
Secondary targets: Duplication removal, improve edge case coverage

Expected overall improvement from addressing high/medium-high priority smells:
- Average complexity: 4.8 → 4.3 (10% reduction)
- Coverage: 92.0% → 94.5%+ (2.5% improvement)
- Duplication: 31 groups → ~25 groups (20% reduction in production duplication)

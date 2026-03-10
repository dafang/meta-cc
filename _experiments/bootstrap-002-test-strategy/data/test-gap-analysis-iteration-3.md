# Test Gap Analysis - Iteration 3

**Date**: 2025-10-18
**Baseline Coverage**: 72.3%

## Summary

Analysis of cmd/ package and error path coverage gaps to guide Iteration 3 test creation.

## cmd/ Package Analysis

**Current Coverage**: 57.9%
**Target Coverage**: 65%+
**Impact**: +2-3% total coverage

### Key Files Needing Tests

#### 1. Root Command (`cmd/root.go`)
**Current**: Minimal testing (Execute() basic test)
**Missing Tests**:
- Version flag handling
- Global flag parsing (--session, --project, --session-only)
- getGlobalOptions() logic with different flag combinations
- Default project path resolution (cwd fallback)
- Error handling for Execute()

**Priority**: HIGH (core CLI entry point)
**Estimated Tests**: 5-6 tests

#### 2. Stats Command (`cmd/stats.go`)
**Current**: Likely untested (no dedicated test file visible)
**Missing Tests**:
- Command execution with valid flags
- Output format handling
- Error cases (invalid session, missing project)

**Priority**: MEDIUM
**Estimated Tests**: 3-4 tests

#### 3. Validate Command (`cmd/validate.go`)
**Current**: Unknown coverage
**Missing Tests**:
- Validation logic for different file types
- Error reporting

**Priority**: MEDIUM
**Estimated Tests**: 2-3 tests

### Functions with 0% Coverage in cmd/

From coverage report:
- `cmd/mcp-server/capabilities.go:128` - CleanupSessionCache (0.0%)
- `cmd/mcp-server/capabilities.go:350` - loadGitHubCapabilities (0.0%)
- `cmd/mcp-server/capabilities.go:864` - readPackageCapability (0.0%)
- `cmd/mcp-server/capabilities.go:947` - readGitHubCapability (0.0%)
- `cmd/mcp-server/logging.go:17` - InitLogger (0.0%)

**Note**: Some of these are infrastructure/initialization functions that are hard to test in isolation. Focus on testable command handlers first.

### Functions with Low Coverage (<60%) in cmd/

- `cmd/mcp-server/capabilities.go:373` - downloadPackage (60.0%) - **error paths missing**
- `cmd/mcp-server/executor.go:36` - ExecuteTool (60.0%) - **error paths missing**
- `cmd/mcp-server/capabilities.go:493` - expandTilde (20.0%) - **missing tests**

## internal/ Package Error Path Gaps

### internal/validation Package (57.9% coverage)
**Current**: Basic validation tests exist
**Missing Error Paths**:
- Invalid URL formats
- Malformed version strings
- Edge cases in description validation
- Null/empty input handling

**Priority**: HIGH (validation is critical)
**Estimated Tests**: 4-5 error path tests

### internal/githelper Package (77.2% coverage)
**Current**: Good coverage but room for error paths
**Missing Error Paths**:
- Git command failures
- Repository not found scenarios
- Permission errors
- Invalid git references

**Priority**: MEDIUM
**Estimated Tests**: 3-4 error path tests

### internal/filter Package (82.1% coverage)
**Current**: Good coverage
**Missing Error Paths**:
- Invalid regex patterns
- Filter composition edge cases

**Priority**: LOW (already good coverage)
**Estimated Tests**: 2 error path tests

### internal/locator Package (81.2% coverage)
**Current**: Good coverage
**Missing Error Paths**:
- File not found scenarios
- Permission denied cases
- Symlink handling

**Priority**: MEDIUM
**Estimated Tests**: 2-3 error path tests

### internal/parser Package (82.1% coverage)
**Current**: Good coverage
**Missing Error Paths**:
- Malformed JSONL
- Truncated input
- Invalid UTF-8

**Priority**: MEDIUM
**Estimated Tests**: 2-3 error path tests

## Recommended Test Creation Plan

### Phase 1: CLI Command Tests (10-12 tests)
Focus on cmd/ package to push from 57.9% → 65%+

1. **root_test.go** (5-6 tests):
   - TestExecute_Success
   - TestExecute_HelpFlag
   - TestExecute_VersionFlag
   - TestGetGlobalOptions_DefaultProjectPath
   - TestGetGlobalOptions_WithSessionFlag
   - TestGetGlobalOptions_WithSessionOnly

2. **stats_test.go** (3-4 tests):
   - TestStatsCommand_ValidSession
   - TestStatsCommand_OutputFormats
   - TestStatsCommand_ErrorHandling
   - TestStatsCommand_Flags

3. **Additional coverage** (2 tests):
   - TestExpandTilde_ErrorPaths (increase from 20%)
   - TestExecuteTool_ErrorPaths (increase from 60%)

**Expected Impact**: +2.5% total coverage

### Phase 2: Error Path Tests (8-10 tests)
Focus on systematic error path coverage

1. **internal/validation** (4-5 tests):
   - Error paths for URL validation
   - Error paths for version validation
   - Edge cases for description validation
   - Null/empty input handling

2. **internal/githelper** (3 tests):
   - Git command failure handling
   - Repository not found
   - Invalid reference handling

3. **internal/parser** (2 tests):
   - Malformed JSONL handling
   - Invalid UTF-8 handling

**Expected Impact**: +1.5% total coverage

## Total Expected Coverage Increase

- **Baseline**: 72.3%
- **CLI tests**: +2.5%
- **Error path tests**: +1.5%
- **Expected Final**: 76.3%

## Success Criteria

- [ ] cmd/ package coverage: 57.9% → 65%+
- [ ] Total coverage: 72.3% → 76%+
- [ ] All new tests pass
- [ ] Test execution time: <150s
- [ ] Pattern library usage documented

## Pattern Usage Guide

For CLI tests:
- Use **Table-Driven Test Pattern** for multiple flag combinations
- Use **Integration Test Pattern** for command execution
- Use **Test Helper Pattern** for test setup

For error path tests:
- Use **Error Path Test Pattern** consistently
- Use **Table-Driven Test Pattern** for multiple error scenarios
- Ensure error messages are validated (not just error presence)

## Time Estimate

- CLI tests: 2.5-3 hours (12 tests × 12-15 min each)
- Error path tests: 1.5-2 hours (10 tests × 10 min each)
- Documentation: 30 min
- **Total**: 4.5-5.5 hours

## Notes

- Focus on high-value tests (command handlers, validation)
- Skip infrastructure/initialization functions (InitLogger, CleanupSessionCache)
- Use existing patterns from pattern library
- Track actual time vs estimated time for V_effectiveness measurement

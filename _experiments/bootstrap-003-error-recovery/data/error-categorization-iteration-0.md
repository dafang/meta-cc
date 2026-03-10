# Error Categorization - Iteration 0

**Date**: 2025-10-18
**Total Errors**: 1336
**Total Tool Calls**: 23103
**Error Rate**: 5.78%

---

## Error Distribution by Tool

| Tool | Error Count | % of Total Errors | Error Category |
|------|-------------|-------------------|----------------|
| Bash | 662 | 49.6% | Command Execution |
| Read | 264 | 19.8% | File Access |
| Edit | 108 | 8.1% | File Modification |
| Write | 42 | 3.1% | File Creation |
| Task | 30 | 2.2% | Request Interruption |
| MCP Tools | 228 | 17.1% | Integration Errors |
| Other | 2 | 0.1% | Miscellaneous |

---

## Error Categories Identified

### Category 1: Build/Compilation Errors (Bash)

**Frequency**: ~200 errors (15% of total)

**Examples**:
- `go: /home/yale/work/meta-cc/go.mod already exists`
- `# github.com/yale/meta-cc/cmd\ncmd/root.go:4:2: "fmt" imported and not used`
- `package command-line-arguments\n\t/tmp/test_signatures.go:5:2: use of internal package not allowed`

**Common Causes**:
- Go compilation errors (syntax, unused imports, type mismatches)
- Module initialization conflicts
- Build path issues

**Impact**: Blocking (prevents code execution)

**Detection**: Parse stderr output for Go compiler messages

---

### Category 2: Test Failures (Bash)

**Frequency**: ~150 errors (11% of total)

**Examples**:
- `--- FAIL: TestLoadFixture (0.00s)\n    fixtures_test.go:34: Fixture content should contain 'sequence' field`
- `FAIL\tgithub.com/yale/meta-cc/internal/testutil\t0.003s`

**Common Causes**:
- Test assertions failing
- Missing test fixtures
- Incorrect test expectations

**Impact**: Blocking (indicates code quality issues)

**Detection**: Parse test output for `FAIL` markers

---

### Category 3: File Not Found (Read/Write)

**Frequency**: ~250 errors (19% of total)

**Examples**:
- `<tool_use_error>File does not exist.</tool_use_error>`
- `wc: /home/yale/work/meta-cc/internal/testutil/fixture.go: No such file or directory`

**Common Causes**:
- Incorrect file paths
- File not created yet
- File moved or deleted

**Impact**: Blocking (cannot proceed without file)

**Detection**: Pattern match for "file not found", "does not exist", "no such file"

---

### Category 4: File Content Size Exceeded (Read)

**Frequency**: ~20 errors (1.5% of total)

**Examples**:
- `File content (46892 tokens) exceeds maximum allowed tokens (25000). Please use offset and limit parameters...`

**Common Causes**:
- Large files exceeding token limits
- Attempting to read entire file without pagination

**Impact**: Recoverable (use offset/limit parameters)

**Detection**: Pattern match for "exceeds maximum allowed tokens"

---

### Category 5: Write Before Read Errors (Write)

**Frequency**: ~40 errors (3% of total)

**Examples**:
- `<tool_use_error>File has not been read yet. Read it first before writing to it.</tool_use_error>`

**Common Causes**:
- Claude Code safety constraint: must read existing files before overwriting
- Workflow violation

**Impact**: Recoverable (read file first, then write)

**Detection**: Pattern match for "File has not been read yet"

---

### Category 6: Command Not Found (Bash)

**Frequency**: ~50 errors (4% of total)

**Examples**:
- `/bin/bash: line 1: meta-cc: command not found`
- `sudo: a terminal is required to read the password`

**Common Causes**:
- Binary not in PATH
- Binary not built yet
- Missing dependencies

**Impact**: Blocking (command unavailable)

**Detection**: Pattern match for "command not found"

---

### Category 7: JSON Parsing Errors (Bash - jq)

**Frequency**: ~80 errors (6% of total)

**Examples**:
- `parse error: Invalid numeric literal at line 1, column 8`
- `jq: error: Cannot index array with string`

**Common Causes**:
- Invalid JSON input
- Incorrect jq filter syntax
- Empty or malformed data

**Impact**: Blocking (data processing fails)

**Detection**: Pattern match for "parse error", "jq: error"

---

### Category 8: Request Interruption (Task)

**Frequency**: ~30 errors (2% of total)

**Examples**:
- `[Request interrupted by user for tool use]`

**Common Causes**:
- User interrupted agent execution
- Claude Code workflow interruption

**Impact**: User-initiated (not a true error)

**Detection**: Pattern match for "Request interrupted"

---

### Category 9: MCP Integration Errors

**Frequency**: ~228 errors (17% of total)

**Examples**:
- MCP tool query failures
- Integration issues with meta-cc MCP server
- Missing capability errors

**Common Causes**:
- MCP server not running
- Query syntax errors
- Missing capabilities

**Impact**: Varies (some queries fail gracefully)

**Detection**: Tool name prefix "mcp__"

---

### Category 10: Permission Denied Errors

**Frequency**: ~10 errors (0.7% of total)

**Examples**:
- `sudo: a terminal is required to read the password; either use the -S option...`

**Common Causes**:
- Insufficient file permissions
- Sudo without interactive terminal

**Impact**: Blocking (cannot perform privileged operation)

**Detection**: Pattern match for "permission denied", "sudo: a password is required"

---

## Error Taxonomy Summary

| Category | Count | % | Impact | Recoverability |
|----------|-------|---|--------|----------------|
| Build/Compilation | 200 | 15.0% | Blocking | Manual fix required |
| Test Failures | 150 | 11.2% | Blocking | Manual fix required |
| File Not Found | 250 | 18.7% | Blocking | Create/fix path |
| File Size Exceeded | 20 | 1.5% | Recoverable | Use pagination |
| Write Before Read | 40 | 3.0% | Recoverable | Read first |
| Command Not Found | 50 | 3.7% | Blocking | Install/build binary |
| JSON Parsing | 80 | 6.0% | Blocking | Fix JSON/jq syntax |
| Request Interruption | 30 | 2.2% | User action | N/A |
| MCP Integration | 228 | 17.1% | Varies | Check MCP server |
| Permission Denied | 10 | 0.7% | Blocking | Fix permissions |
| **Other/Uncategorized** | 278 | 20.9% | Varies | Case-by-case |

**Coverage**: 10 categories covering ~79% of errors (1058/1336)

---

## Key Insights

1. **Bash errors dominate** (49.6%): Most errors are command execution failures (build, test, file ops)
2. **File access issues common** (19.8%): File not found, file not read yet
3. **MCP integration errors significant** (17.1%): Many errors from MCP tool queries
4. **Error rate is high** (5.78%): Nearly 1 in 17 tool calls fails
5. **Most errors are blocking**: Require manual intervention to resolve

---

## Baseline Metrics

- **Overall Error Rate**: 5.78%
- **Error Categories Identified**: 10
- **Category Coverage**: 79.1%
- **Most Error-Prone Tool**: Bash (662 errors, 49.6%)
- **Mean Time To Diagnosis (MTTD)**: Unknown (baseline)
- **Mean Time To Recovery (MTTR)**: Unknown (baseline)
- **Recovery Success Rate**: Unknown (baseline)
- **Error Detection Coverage**: ~50% (manual observation only)

---

## Existing Error Handling Patterns

**Current Approach**: Ad-hoc manual error diagnosis and recovery

**Observed Patterns**:
1. **Retry after fix**: Fix code, rerun command
2. **Read before write**: Add Read call before Write
3. **Path correction**: Fix file paths when "file not found"
4. **Syntax fix**: Correct Go/jq syntax errors
5. **Ignore and continue**: Some errors ignored (MCP timeouts)

**Gaps**:
- No systematic error classification
- No automated error detection
- No documented recovery procedures
- No preventive measures in place
- No error trend analysis
- No error rate monitoring

---

## Next Steps (Iteration 1 Focus)

1. Create comprehensive error classification taxonomy (expand to 12+ categories)
2. Document root cause diagnosis procedures for top 5 error types
3. Define recovery strategy patterns for common errors
4. Establish baseline MTTD and MTTR measurements
5. Implement error pattern detection (automated)

---

**Generated**: 2025-10-18
**Data Source**: meta-cc MCP query-tools (project scope, 1336 errors)
**Analysis Method**: Manual categorization of error samples

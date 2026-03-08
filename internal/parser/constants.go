package parser

// MaxScannerLineBytes is the maximum line size for bufio.Scanner when reading
// JSONL session files. Both the session parser and the MCP query executor
// share this constant to ensure consistent handling of large lines.
const MaxScannerLineBytes = 4 * 1024 * 1024 // 4 MB

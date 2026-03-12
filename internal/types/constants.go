package types

// LargeLineWarnBytes is the soft warning threshold for large lines when reading
// JSONL session files. Lines exceeding this size trigger a debug log but are
// still processed normally — this is purely observational, not a hard limit.
// Both the session parser and the MCP query executor reference this constant.
// TODO(post-stabilization): remove LargeLineWarnBytes once streaming reader is proven stable
const LargeLineWarnBytes = 4 * 1024 * 1024 // 4 MB

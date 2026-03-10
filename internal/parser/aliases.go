package parser

import "github.com/yaleh/meta-cc/internal/types"

// Type aliases for backward compatibility.
// Domain types now live in internal/types; these aliases allow existing
// consumers to keep using parser.SessionEntry etc. without modification.

type SessionEntry = types.SessionEntry
type Message = types.Message
type ContentBlock = types.ContentBlock
type ToolUse = types.ToolUse
type ToolResult = types.ToolResult
type ToolCall = types.ToolCall

// MaxScannerLineBytes is an alias for the constant in internal/types.
const MaxScannerLineBytes = types.MaxScannerLineBytes

// ExtractToolCalls is an alias for the function in internal/types.
var ExtractToolCalls = types.ExtractToolCalls

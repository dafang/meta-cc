package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/yaleh/meta-cc/internal/types"
)

// FilterStrategy controls how ReadLineFiltered handles image content.
type FilterStrategy int

const (
	// StrategyDefault truncates base64 image data in-place (Strategy B).
	StrategyDefault FilterStrategy = iota
	// StrategySkipImage skips the entire line when "type":"image" is present (Strategy A).
	StrategySkipImage
)

// ReadLineFiltered reads one line from r and applies the given FilterStrategy.
// Returns (line, skipped, error).
//   - skipped=true means the line was intentionally omitted (StrategySkipImage).
//   - On EOF with remaining data, returns the data with io.EOF.
func ReadLineFiltered(r *bufio.Reader, strategy FilterStrategy) ([]byte, bool, error) {
	line, err := r.ReadBytes('\n')

	// ReadBytes returns err=io.EOF if the final "line" has no trailing newline.
	// We still want to process any data returned before propagating EOF.
	if len(line) == 0 && err != nil {
		return nil, false, err
	}

	// Observational monitoring: log large lines at debug level (not an error).
	if len(line) > types.LargeLineWarnBytes {
		slog.Debug("large line detected", "bytes", len(line))
	}

	switch strategy {
	case StrategySkipImage:
		if bytes.Contains(line, []byte(`"type":"image"`)) {
			return nil, true, err
		}
		return line, false, err

	default: // StrategyDefault
		processed, valid := stripImageData(line)
		if !valid {
			// Fallback: return original line unmodified
			return line, false, err
		}
		return processed, false, err
	}
}

// stripImageData replaces base64 "source.data" values inside image blocks with
// "<binary-omitted>". It loops until no `"type":"base64"` remains in the line.
// Returns (processedLine, valid):
//   - valid=false means the result is not valid JSON; caller should use original.
func stripImageData(line []byte) ([]byte, bool) {
	// Fast path: no image content detected.
	if !bytes.Contains(line, []byte(`"type":"image"`)) ||
		!bytes.Contains(line, []byte(`"type":"base64"`)) {
		return line, true
	}

	current := replaceAllDataFields(line)

	if !json.Valid(current) {
		return line, false
	}
	return current, true
}

// replaceAllDataFields replaces every `"data":"<value>"` occurrence in line
// with `"data":"<binary-omitted>"` in a single left-to-right pass.
func replaceAllDataFields(line []byte) []byte {
	const prefix = `"data":"`
	prefixBytes := []byte(prefix)

	var buf bytes.Buffer
	remaining := line

	for {
		idx := bytes.Index(remaining, prefixBytes)
		if idx == -1 {
			buf.Write(remaining)
			break
		}

		// Write up to and including the prefix
		buf.Write(remaining[:idx+len(prefixBytes)])

		// Advance past prefix to find the closing quote of the value
		rest := remaining[idx+len(prefixBytes):]
		end := 0
		for end < len(rest) {
			if rest[end] == '"' {
				break
			}
			// Handle escaped characters inside the string
			if rest[end] == '\\' && end+1 < len(rest) {
				end += 2
				continue
			}
			end++
		}
		if end >= len(rest) {
			// No closing quote — write rest as-is and stop
			buf.Write(rest)
			break
		}

		// Replace value with placeholder; write closing quote
		buf.WriteString("<binary-omitted>")
		buf.WriteByte('"') // closing quote

		// Advance past the closing quote
		remaining = rest[end+1:]
	}

	return buf.Bytes()
}

// ReadAllFiltered reads all lines from r using ReadLineFiltered and returns
// the raw JSON bytes of non-empty lines. It is a convenience wrapper used
// by the stage2 executors.
func ReadAllFiltered(r *bufio.Reader, strategy FilterStrategy) ([]json.RawMessage, error) {
	var results []json.RawMessage
	for {
		line, _, err := ReadLineFiltered(r, strategy)
		trimmed := bytes.TrimRight(line, "\r\n")
		if len(bytes.TrimSpace(trimmed)) > 0 {
			cp := make([]byte, len(trimmed))
			copy(cp, trimmed)
			results = append(results, json.RawMessage(cp))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

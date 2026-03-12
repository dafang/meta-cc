package parser

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

// buildImageJSONLLine builds a realistic JSONL line containing an image block with given base64 data.
func buildImageJSONLLine(b64data string) string {
	return fmt.Sprintf(
		`{"type":"user","uuid":"test-uuid-001","message":{"role":"user","content":[{"type":"tool_result","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"%s"}}]}]}}`,
		b64data,
	)
}

// buildTwoImageJSONLLine builds a JSONL line with two image blocks.
func buildTwoImageJSONLLine(b64a, b64b string) string {
	return fmt.Sprintf(
		`{"type":"user","uuid":"test-uuid-002","message":{"role":"user","content":[{"type":"tool_result","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"%s"}},{"type":"image","source":{"type":"base64","media_type":"image/jpeg","data":"%s"}}]}]}}`,
		b64a, b64b,
	)
}

// --- stripImageData tests ---

func TestStripImageData_PlainLine(t *testing.T) {
	line := []byte(`{"type":"user","uuid":"abc","message":{"role":"user","content":"hello"}}`)
	got, valid := stripImageData(line)
	if !valid {
		t.Error("expected valid=true for plain line")
	}
	if !bytes.Equal(got, line) {
		t.Errorf("expected line unchanged, got: %s", got)
	}
}

func TestStripImageData_TextToolResult(t *testing.T) {
	line := []byte(`{"type":"user","uuid":"abc","message":{"role":"user","content":[{"type":"tool_result","content":[{"type":"text","text":"some output"}]}]}}`)
	got, valid := stripImageData(line)
	if !valid {
		t.Error("expected valid=true for text tool_result")
	}
	if !bytes.Equal(got, line) {
		t.Errorf("expected line unchanged for text tool_result, got: %s", got)
	}
}

func TestStripImageData_SingleImage(t *testing.T) {
	b64data := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("A"), 1024))
	line := []byte(buildImageJSONLLine(b64data))
	got, valid := stripImageData(line)
	if !valid {
		t.Errorf("expected valid=true after stripping, got invalid JSON")
	}
	if !json.Valid(got) {
		t.Errorf("stripped result is not valid JSON: %s", got)
	}
	if bytes.Contains(got, []byte(b64data)) {
		t.Error("base64 data should have been replaced")
	}
	if !bytes.Contains(got, []byte("<binary-omitted>")) {
		t.Error("expected <binary-omitted> placeholder in output")
	}
}

func TestStripImageData_MultipleImages(t *testing.T) {
	b64a := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("B"), 512))
	b64b := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("C"), 256))
	line := []byte(buildTwoImageJSONLLine(b64a, b64b))
	got, valid := stripImageData(line)
	if !valid {
		t.Error("expected valid=true after stripping two images")
	}
	if !json.Valid(got) {
		t.Errorf("stripped result is not valid JSON: %s", got)
	}
	if bytes.Contains(got, []byte(b64a)) || bytes.Contains(got, []byte(b64b)) {
		t.Error("both base64 data values should have been replaced")
	}
	// Count occurrences of placeholder
	count := bytes.Count(got, []byte("<binary-omitted>"))
	if count < 2 {
		t.Errorf("expected at least 2 <binary-omitted> placeholders, got %d", count)
	}
}

func TestStripImageData_NonBase64Image(t *testing.T) {
	// type=image but source type is url, not base64
	line := []byte(`{"type":"user","uuid":"abc","message":{"role":"user","content":[{"type":"tool_result","content":[{"type":"image","source":{"type":"url","url":"https://example.com/img.png"}}]}]}}`)
	got, valid := stripImageData(line)
	if !valid {
		t.Error("expected valid=true for non-base64 image")
	}
	if !bytes.Equal(got, line) {
		t.Errorf("expected line unchanged for non-base64 image")
	}
}

func TestStripImageData_AfterStrip_ValidJSON(t *testing.T) {
	// 5KB base64 image data
	rawData := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 1024)
	b64data := base64.StdEncoding.EncodeToString(rawData)
	line := []byte(buildImageJSONLLine(b64data))
	got, valid := stripImageData(line)
	if !valid {
		t.Error("expected valid=true")
	}
	if !json.Valid(got) {
		t.Errorf("stripped result must be valid JSON, got: %.200s", got)
	}
}

func TestStripImageData_InvalidAfterStrip_Fallback(t *testing.T) {
	// Craft a line where "data":"..." replacement would break JSON structure.
	// We simulate by passing a line that has type=image and type=base64 markers
	// but in a malformed way that json.Valid would reject after naive replacement.
	// Use a line where the "data" field is followed by extra quote that creates invalid JSON.
	// Actually, let's build a synthetic test: inject a raw line that looks like image
	// but after replacement becomes invalid (inject a line that's already not fully valid JSON
	// except for the data field).
	// Easiest: build normal image line, then corrupt JSON outside data field.
	b64data := base64.StdEncoding.EncodeToString([]byte("test"))
	// Build malformed JSON that contains image markers but isn't valid JSON after stripping:
	malformed := []byte(fmt.Sprintf(
		`{"type":"image","source":{"type":"base64","data":"%s"} BROKEN`,
		b64data,
	))
	got, valid := stripImageData(malformed)
	if valid {
		t.Error("expected valid=false when JSON is broken after strip")
	}
	// Should return original
	if !bytes.Equal(got, malformed) {
		t.Error("on invalid result, should return original line")
	}
}

// --- ReadLineFiltered tests ---

func TestReadLineFiltered_StrategyB_LargeImage(t *testing.T) {
	// 5MB image line
	rawData := bytes.Repeat([]byte{0xAB, 0xCD}, 2*1024*1024) // ~4MB binary → ~5.4MB base64
	b64data := base64.StdEncoding.EncodeToString(rawData)
	line := buildImageJSONLLine(b64data)
	input := line + "\n"

	r := bufio.NewReader(strings.NewReader(input))
	got, skipped, err := ReadLineFiltered(r, StrategyDefault)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Error("StrategyDefault should not skip image lines, only truncate")
	}
	if !json.Valid(got) {
		t.Errorf("output should be valid JSON after stripping, len=%d, preview=%.200s", len(got), got)
	}
	if bytes.Contains(got, []byte(b64data)) {
		t.Error("large base64 data should have been stripped")
	}
}

func TestReadLineFiltered_StrategyA_ImageLine(t *testing.T) {
	b64data := base64.StdEncoding.EncodeToString([]byte("small image data"))
	line := buildImageJSONLLine(b64data) + "\n"

	r := bufio.NewReader(strings.NewReader(line))
	_, skipped, err := ReadLineFiltered(r, StrategySkipImage)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Error("StrategySkipImage should skip image lines")
	}
}

func TestReadLineFiltered_StrategyA_TextLine(t *testing.T) {
	line := `{"type":"user","uuid":"abc","message":{"role":"user","content":"hello"}}` + "\n"

	r := bufio.NewReader(strings.NewReader(line))
	got, skipped, err := ReadLineFiltered(r, StrategySkipImage)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Error("StrategySkipImage should not skip non-image lines")
	}
	trimmed := string(bytes.TrimRight(got, "\n"))
	expected := strings.TrimRight(line, "\n")
	if trimmed != expected {
		t.Errorf("expected line content unchanged, got: %s", trimmed)
	}
}

func TestReadLineFiltered_EmptyLine(t *testing.T) {
	input := "\n"
	r := bufio.NewReader(strings.NewReader(input))
	got, skipped, err := ReadLineFiltered(r, StrategyDefault)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Error("empty line should not be skipped (skipped=false)")
	}
	trimmed := bytes.TrimRight(got, "\n")
	if len(trimmed) != 0 {
		t.Errorf("expected empty content, got: %q", trimmed)
	}
}

func TestReadLineFiltered_NoTrailingNewline(t *testing.T) {
	line := `{"type":"user","uuid":"xyz","message":{"role":"user","content":"no newline"}}`
	r := bufio.NewReader(strings.NewReader(line))
	got, skipped, err := ReadLineFiltered(r, StrategyDefault)
	if err != io.EOF {
		t.Errorf("expected io.EOF for last line without newline, got: %v", err)
	}
	if skipped {
		t.Error("should not skip non-image line")
	}
	if string(bytes.TrimRight(got, "\n")) != line {
		t.Errorf("expected content %q, got %q", line, bytes.TrimRight(got, "\n"))
	}
}

func TestReadLineFiltered_NormalLineAfterLargeLine(t *testing.T) {
	// Large image line followed by a normal line
	rawData := bytes.Repeat([]byte{0x01}, 1*1024*1024) // 1MB binary → ~1.3MB base64
	b64data := base64.StdEncoding.EncodeToString(rawData)
	imageLine := buildImageJSONLLine(b64data)

	normalUUID := "normal-line-uuid-9876"
	normalLine := fmt.Sprintf(`{"type":"user","uuid":"%s","message":{"role":"user","content":"after large"}}`, normalUUID)

	input := imageLine + "\n" + normalLine + "\n"
	r := bufio.NewReaderSize(strings.NewReader(input), 64*1024) // small internal buffer to stress test

	// Read first line (image)
	firstGot, firstSkipped, firstErr := ReadLineFiltered(r, StrategyDefault)
	if firstErr != nil && firstErr != io.EOF {
		t.Fatalf("error reading first line: %v", firstErr)
	}
	if firstSkipped {
		t.Error("first line should not be skipped with StrategyDefault")
	}
	if !json.Valid(firstGot) {
		t.Errorf("first line should be valid JSON after stripping")
	}

	// Read second line (normal)
	secondGot, secondSkipped, secondErr := ReadLineFiltered(r, StrategyDefault)
	if secondErr != nil && secondErr != io.EOF {
		t.Fatalf("error reading second line: %v", secondErr)
	}
	if secondSkipped {
		t.Error("second line should not be skipped")
	}
	if !bytes.Contains(secondGot, []byte(normalUUID)) {
		t.Errorf("second line should contain UUID %s, got: %.200s", normalUUID, secondGot)
	}
}

// TestReadLineFiltered_LargeLineMonitoring verifies that lines exceeding
// LargeLineWarnBytes do NOT return an error — monitoring is purely observational.
func TestReadLineFiltered_LargeLineMonitoring(t *testing.T) {
	// Build a line larger than LargeLineWarnBytes (4MB) by padding base64 data.
	rawData := bytes.Repeat([]byte{0xAB}, 3*1024*1024) // 3MB binary → ~4MB base64
	b64data := base64.StdEncoding.EncodeToString(rawData)
	largeLine := buildImageJSONLLine(b64data)

	r := bufio.NewReaderSize(strings.NewReader(largeLine+"\n"), 64*1024)

	got, skipped, err := ReadLineFiltered(r, StrategyDefault)
	if err != nil && err != io.EOF {
		t.Fatalf("large line must not cause an error, got: %v", err)
	}
	if skipped {
		t.Error("large line should not be skipped with StrategyDefault")
	}
	if len(got) == 0 {
		t.Error("large line should produce non-empty output")
	}
}

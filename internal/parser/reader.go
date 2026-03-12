package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// SessionParser 负责解析 Claude Code 会话文件
type SessionParser struct {
	filePath string
}

// NewSessionParser 创建 SessionParser 实例
func NewSessionParser(filePath string) *SessionParser {
	return &SessionParser{
		filePath: filePath,
	}
}

// ParseEntries 解析 JSONL 文件，返回 SessionEntry 数组
// JSONL 格式：每行一个 JSON 对象
// 处理规则：
//   - 跳过空行和空白行
//   - 非法 JSON 行返回错误
//   - 仅返回消息类型（type == "user" 或 "assistant"）
//   - 过滤掉 file-history-snapshot 等非消息类型
func (p *SessionParser) ParseEntries() ([]SessionEntry, error) {
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	var entries []SessionEntry
	r := bufio.NewReader(file)
	lineNum := 0

	for {
		line, skipped, err := ReadLineFiltered(r, StrategyDefault)
		if skipped || len(bytes.TrimSpace(line)) == 0 {
			if err == io.EOF {
				break
			}
			continue
		}

		lineNum++

		// 解析 JSON 为 SessionEntry
		var entry SessionEntry
		if jsonErr := json.Unmarshal(bytes.TrimRight(line, "\n"), &entry); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, jsonErr)
		}

		// 仅保留消息类型
		if entry.IsMessage() {
			entries = append(entries, entry)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading session file: %w", err)
		}
	}

	return entries, nil
}

// ParseEntriesFromContent 从字符串内容解析 JSONL（用于测试）
func ParseEntriesFromContent(content string) ([]SessionEntry, error) {
	var entries []SessionEntry
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// 跳过空行
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry SessionEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum+1, err)
		}

		// 仅保留消息类型
		if entry.IsMessage() {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

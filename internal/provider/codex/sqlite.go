package codex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yaleh/meta-cc/internal/conversation"
)

func listSessionsFromDB(ctx context.Context, dsn string) ([]conversation.Session, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := ensureThreadsTable(ctx, db); err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, "SELECT * FROM threads ORDER BY created_at DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []conversation.Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func getSessionFromDB(ctx context.Context, dsn, sessionID string) (conversation.Session, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return conversation.Session{}, err
	}
	defer db.Close()

	if err := ensureThreadsTable(ctx, db); err != nil {
		return conversation.Session{}, err
	}

	rows, err := db.QueryContext(ctx, "SELECT * FROM threads WHERE id = ? LIMIT 1", sessionID)
	if err != nil {
		return conversation.Session{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return conversation.Session{}, ErrSessionNotFound
	}

	return scanSession(rows)
}

func ensureThreadsTable(ctx context.Context, db *sql.DB) error {
	var name string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='threads'").Scan(&name)
	if err == sql.ErrNoRows || name == "" {
		return &SQLiteSchemaError{Message: "missing threads table"}
	}
	return err
}

func scanSession(rows *sql.Rows) (conversation.Session, error) {
	cols, err := rows.Columns()
	if err != nil {
		return conversation.Session{}, err
	}
	values := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return conversation.Session{}, err
	}

	colMap := make(map[string]interface{}, len(cols))
	for i, col := range cols {
		colMap[col] = values[i]
	}

	required := []string{"id", "cwd", "title", "source"}
	for _, key := range required {
		if _, ok := colMap[key]; !ok {
			return conversation.Session{}, &SQLiteSchemaError{Message: "missing column: " + key}
		}
	}

	createdAt := extractTime(colMap["created_at_ms"])
	if createdAt.IsZero() {
		createdAt = extractTime(colMap["created_at"])
	}

	ext, _ := json.Marshal(map[string]interface{}{
		"rollout_path":   stringValue(colMap["rollout_path"]),
		"source":         stringValue(colMap["source"]),
		"thread_source":  stringValue(colMap["thread_source"]),
		"model_provider": stringValue(colMap["model_provider"]),
	})

	return conversation.Session{
		ID:        stringValue(colMap["id"]),
		Provider:  conversation.ProviderCodex,
		Title:     stringValue(colMap["title"]),
		CWD:       stringValue(colMap["cwd"]),
		Model:     firstNonEmpty(stringValue(colMap["model"]), stringValue(colMap["model_provider"])),
		CreatedAt: createdAt.UTC(),
		TokenUsage: conversation.TokenUsage{
			InputTokens: intValue(colMap["tokens_used"]),
		},
		Extensions: ext,
	}, nil
}

func extractTime(value interface{}) time.Time {
	switch v := value.(type) {
	case int64:
		if v > 1_000_000_000_000 {
			return time.UnixMilli(v)
		}
		return time.Unix(v, 0)
	case []byte:
		return extractTime(stringValue(v))
	case string:
		var iv int64
		_, _ = fmt.Sscan(v, &iv)
		return extractTime(iv)
	default:
		return time.Time{}
	}
}

func stringValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return ""
	}
}

func intValue(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case []byte:
		var n int
		_, _ = fmt.Sscan(string(x), &n)
		return n
	case string:
		var n int
		_, _ = fmt.Sscan(x, &n)
		return n
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

package codex

import (
	"errors"
	"fmt"
)

var ErrSessionNotFound = errors.New("session not found")

type SQLiteSchemaError struct {
	Message string
}

func (e *SQLiteSchemaError) Error() string {
	return fmt.Sprintf("codex sqlite schema error: %s", e.Message)
}

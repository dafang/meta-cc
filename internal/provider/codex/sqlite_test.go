package codex

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteListAndGetSession(t *testing.T) {
	db, err := sql.Open("sqlite", "file:test-sqlite?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE threads (
		id TEXT PRIMARY KEY,
		rollout_path TEXT,
		cwd TEXT,
		title TEXT,
		model TEXT,
		model_provider TEXT,
		tokens_used INTEGER,
		source TEXT,
		created_at INTEGER
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO threads(id, rollout_path, cwd, title, model, model_provider, tokens_used, source, created_at)
	VALUES ('s1', '/tmp/rollout.jsonl', '/tmp', 'hello', 'gpt-5', 'openai', 7, 'cli', 1700000000)`)
	if err != nil {
		t.Fatal(err)
	}

	sessions, err := listSessionsFromDB(context.Background(), "file:test-sqlite?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("listSessionsFromDB: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "s1" {
		t.Fatalf("unexpected sessions: %#v", sessions)
	}

	session, err := getSessionFromDB(context.Background(), "file:test-sqlite?mode=memory&cache=shared", "s1")
	if err != nil {
		t.Fatalf("getSessionFromDB: %v", err)
	}
	if session.Title != "hello" {
		t.Fatalf("unexpected session: %#v", session)
	}
}

func TestSQLiteMissingTableAndUnknownSession(t *testing.T) {
	db, err := sql.Open("sqlite", "file:test-missing?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = listSessionsFromDB(context.Background(), "file:test-missing?mode=memory&cache=shared")
	var schemaErr *SQLiteSchemaError
	if !errors.As(err, &schemaErr) {
		t.Fatalf("expected SQLiteSchemaError, got %v", err)
	}

	_, err = db.Exec(`CREATE TABLE threads (id TEXT PRIMARY KEY, cwd TEXT, title TEXT, source TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = getSessionFromDB(context.Background(), "file:test-missing?mode=memory&cache=shared", "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

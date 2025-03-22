package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const _table = "less.record"

type record struct {
	key, val string
	deadline time.Time
}

type SqliteStorage struct {
	db *sql.DB
}

func New(db *sql.DB) *SqliteStorage {
	return &SqliteStorage{
		db: db,
	}
}

func (sq *SqliteStorage) Renew(ctx context.Context, key string, deadline time.Time) error {
	_, err := sq.db.ExecContext(ctx, `UPDATE less_record SET deadline = ? WHERE key = ?`, deadline, key)
	if err != nil {
		return err
	}

	return nil
}

func (sq *SqliteStorage) Get(ctx context.Context, key string) (string, error) {
	row := sq.db.QueryRowContext(ctx, `SELECT key, value, deadline FROM less_record WHERE key = ?`, key)

	var record record
	err := row.Scan(&record.key, &record.val, &record.deadline)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if record.deadline.Before(time.Now()) {
		return "", nil
	}

	return record.val, nil
}

func (sq *SqliteStorage) SetNX(ctx context.Context, key string, val string, deadline time.Time) (bool, error) {
	row := sq.db.QueryRowContext(ctx,
		`INSERT INTO less_record(key, value, deadline) VALUES (?, ?, ?)
                ON CONFLICT (key) DO UPDATE SET 
                        value = EXCLUDED.value,
                        deadline = EXCLUDED.deadline
                WHERE 
                        less_record.deadline < ?
                RETURNING value`, key, val, deadline, time.Now())

	var set string
	err := row.Scan(&set)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

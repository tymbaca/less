package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type record struct {
	key, val string
	deadline time.Time
}

type PostgresStorage struct {
	db *sql.DB
}

// New creates sqlite3 storage. If you have a lot of candidates (50 and
// more) it's recomended to do db.SetMaxOpenConns(1) before calling New.
func New(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{
		db: db,
	}
}

func (sq *PostgresStorage) Renew(ctx context.Context, key string, deadline time.Time) error {
	_, err := sq.db.ExecContext(ctx, `UPDATE less.record SET deadline = $1 WHERE key = $2`, deadline, key)
	if err != nil {
		return err
	}

	return nil
}

func (sq *PostgresStorage) Get(ctx context.Context, key string) (string, error) {
	row := sq.db.QueryRowContext(ctx, `SELECT key, value, deadline FROM less.record WHERE key = $1`, key)

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

func (sq *PostgresStorage) SetNX(ctx context.Context, key string, val string, deadline time.Time) (bool, error) {
	row := sq.db.QueryRowContext(ctx,
		`INSERT INTO less.record(key, value, deadline) VALUES ($1, $2, $3)
                ON CONFLICT (key) DO UPDATE SET 
                        value = EXCLUDED.value,
                        deadline = EXCLUDED.deadline
                WHERE 
                        less.record.deadline < $4
                RETURNING value`,
		key, val, deadline, time.Now())

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

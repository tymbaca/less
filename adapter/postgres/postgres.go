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
	_, err := sq.db.ExecContext(ctx, `UPDATE less.keys SET deadline = $1 WHERE key = $2`, deadline, key)
	if err != nil {
		return err
	}

	return nil
}

func (sq *PostgresStorage) Get(ctx context.Context, key string) (string, error) {
	row := sq.db.QueryRowContext(ctx, `SELECT key, node, deadline FROM less.keys WHERE key = $1`, key)

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
		`INSERT INTO less.keys(key, node, deadline) VALUES ($1, $2, $3)
                ON CONFLICT (key) DO UPDATE SET 
                        node = EXCLUDED.node,
                        deadline = EXCLUDED.deadline
                WHERE 
                        less.keys.deadline < $4
                RETURNING node`,
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

func (p *PostgresStorage) List(ctx context.Context, keys []string) (map[string]string, error) {
	panic("not implemented") // TODO: lists active records in keys table
}

func (p *PostgresStorage) ListNodes(ctx context.Context) ([]string, error) {
	panic("not implemented") // TODO: list all active nodes (by comparing time.Now with rows deadline)
}

func (p *PostgresStorage) RenewLiveness(ctx context.Context, id string, deadline time.Time) error {
	panic("not implemented") // TODO: inserts or updates deadline
}

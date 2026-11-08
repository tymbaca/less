package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

func (sq *PostgresStorage) List(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	var keysBuilder strings.Builder
	keysBuilder.WriteString("(")
	args := make([]any, 0, len(keys))
	for i, key := range keys {
		if i > 0 {
			keysBuilder.WriteString(", ")
		}
		fmt.Fprintf(&keysBuilder, "$%d", i+1)
		args = append(args, key)
	}
	keysBuilder.WriteString(")")
	keysSQL := keysBuilder.String()

	rows, err := sq.db.QueryContext(ctx,
		`SELECT key, node, deadline FROM less.keys WHERE key IN `+keysSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	now := time.Now()
	for rows.Next() {
		var record record
		err := rows.Scan(&record.key, &record.val, &record.deadline)
		if err != nil {
			return nil, err
		}

		if record.deadline.Before(now) {
			continue
		}

		result[record.key] = record.val
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (sq *PostgresStorage) ListNodes(ctx context.Context) ([]string, error) {
	rows, err := sq.db.QueryContext(ctx,
		`SELECT node FROM less.nodes WHERE deadline > $1`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []string
	for rows.Next() {
		var node string
		if err := rows.Scan(&node); err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (sq *PostgresStorage) RenewLiveness(ctx context.Context, node string, deadline time.Time) error {
	_, err := sq.db.ExecContext(ctx,
		`INSERT INTO less.nodes(node, deadline) VALUES ($1, $2)
                ON CONFLICT (node) DO UPDATE SET 
                        deadline = EXCLUDED.deadline`,
		node, deadline)
	if err != nil {
		return err
	}

	return nil
}

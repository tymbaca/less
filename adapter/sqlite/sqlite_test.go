package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/tymbaca/less"
)

func TestSqlite(t *testing.T) {
	db, err := sql.Open("sqlite3", "testdata/test.db")
	require.NoError(t, err)

	migr, err := os.ReadFile("migration.up.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(migr))
	require.NoError(t, err)

	ctx := context.TODO()
	storage := New(db)

	stor1 := wrap(storage)
	cand1 := less.New(ctx, stor1, less.WithApplyRate(50*time.Millisecond), less.WithRenewRate(50*time.Millisecond), less.WithTTL(200*time.Millisecond))

	time.Sleep(100 * time.Millisecond)

	stor2 := wrap(storage)
	cand2 := less.New(ctx, stor2, less.WithApplyRate(50*time.Millisecond), less.WithRenewRate(50*time.Millisecond), less.WithTTL(200*time.Millisecond))

	require.True(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())

	time.Sleep(300 * time.Millisecond)

	// same
	require.True(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())

	stor1.shotdown(300 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	require.False(t, cand1.IsLeader())
	require.True(t, cand2.IsLeader())

	stor2.shotdown(300 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	require.True(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())
}

type shotdownWrapper struct {
	s    less.Storage
	down bool
}

func wrap(s less.Storage) *shotdownWrapper {
	return &shotdownWrapper{s: s, down: false}
}

func (wr *shotdownWrapper) shotdown(dur time.Duration) {
	go func() {
		wr.down = true
		<-time.After(dur)
		wr.down = false
	}()
}

func (wr *shotdownWrapper) Renew(ctx context.Context, key string, deadline time.Time) error {
	if wr.down {
		return errors.New("node is down")
	}

	return wr.s.Renew(ctx, key, deadline)
}

func (wr *shotdownWrapper) Get(ctx context.Context, key string) (string, error) {
	if wr.down {
		return "", errors.New("node is down")
	}

	return wr.s.Get(ctx, key)
}

func (wr *shotdownWrapper) SetNX(ctx context.Context, key string, val string, deadline time.Time) (bool, error) {
	if wr.down {
		return false, errors.New("node is down")
	}

	return wr.s.SetNX(ctx, key, val, deadline)
}

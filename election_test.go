package less

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	"github.com/tymbaca/less/adapter/postgres"
	"github.com/tymbaca/less/adapter/sqlite"
)

var opts = []Option{WithFollowRate(50 * time.Millisecond), WithHoldRate(50 * time.Millisecond), WithTTL(200 * time.Millisecond)}

func Example() {
	ctx := context.Background()

	var storage Storage
	// Fill with any shared storage implementation
	// You can also use our adapters from `adapter` package

	candidate := New(ctx, storage, WithKey("notify-users"))

	workFn := func() {
		if !candidate.IsLeader() {
			return
		}
		// do work
		fmt.Println("notification sent")
	}

	for range time.Tick(1 * time.Minute) {
		workFn()
	}

	// Now if we run this code on 2 and more nodes, we can be sure that
	// only one of them will actively run the job
}

func TestSqlite(t *testing.T) {
	db, err := sql.Open("sqlite3", "testdata/test.db")
	require.NoError(t, err)

	db.SetMaxOpenConns(1)

	migr, err := os.ReadFile("adapter/sqlite/migration.up.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(migr))
	require.NoError(t, err)

	storage := sqlite.New(db)

	t.Run("testGeneral", func(t *testing.T) {
		_, err := db.Exec("DELETE FROM less_record")
		require.NoError(t, err)
		testGeneral(t, storage)
	})
	t.Run("testALotOfCandidates", func(t *testing.T) {
		_, err := db.Exec("DELETE FROM less_record")
		require.NoError(t, err)
		testALotOfCandidates(t, storage, 100)
	})
	t.Run("testALotOfWorkers", func(t *testing.T) {
		_, err := db.Exec("DELETE FROM less_record")
		require.NoError(t, err)
		testALotOfWorkers(t, storage, 100)
	})
}

func TestPostgres(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		t.Skip("postgres in not up")
	}
	conn.Close()

	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	require.NoError(t, err)

	migr, err := os.ReadFile("adapter/postgres/migration.up.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(migr))
	require.NoError(t, err)

	storage := postgres.New(db)

	t.Run("testGeneral", func(t *testing.T) {
		_, err := db.Exec("DELETE FROM less.record")
		require.NoError(t, err)
		testGeneral(t, storage)
	})
	t.Run("testALotOfCandidates", func(t *testing.T) {
		_, err := db.Exec("DELETE FROM less.record")
		require.NoError(t, err)
		testALotOfCandidates(t, storage, 200)
	})
	t.Run("testALotOfWorkers", func(t *testing.T) {
		_, err := db.Exec("DELETE FROM less.record")
		require.NoError(t, err)
		testALotOfWorkers(t, storage, 200)
	})
}

func testGeneral(t *testing.T, storage Storage) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stor1 := wrap(storage)
	stor2 := wrap(storage)

	// Launch the first candidate
	cand1 := New(ctx, stor1, opts...)

	time.Sleep(100 * time.Millisecond)

	// Launch the second candidate
	cand2 := New(ctx, stor2, opts...)

	// first already got leadership
	require.True(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())

	time.Sleep(300 * time.Millisecond)

	// same
	require.True(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())

	// first candidate hangs, it will lose leadership
	stor1.shutdown(300 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	// second candidate get's the leadership
	require.False(t, cand1.IsLeader())
	require.True(t, cand2.IsLeader())

	// vice-versa
	stor2.shutdown(300 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	require.True(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())

	cancel()
	time.Sleep(100 * time.Millisecond)

	// both must exit and release the leadership after ctx is canceled
	require.False(t, cand1.IsLeader())
	require.False(t, cand2.IsLeader())
}

func testALotOfCandidates(t *testing.T, storage Storage, count int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cands []*Candidate

	for range count {
		cands = append(cands, New(ctx, storage, opts...))
	}

	time.Sleep(100 * time.Millisecond)

	t.Run("only one candidate is leader", func(t *testing.T) {
		leaders := 0
		for _, cand := range cands {
			if cand.IsLeader() {
				leaders++
			}
		}
		require.Equal(t, 1, leaders)
	})

	t.Run("if we cancel the context, there will be no leaders", func(t *testing.T) {
		cancel()
		time.Sleep(100 * time.Millisecond)

		leaders := 0
		for _, cand := range cands {
			if cand.IsLeader() {
				leaders++
			}
		}
		require.Equal(t, 0, leaders)
	})
}

func testALotOfWorkers(t *testing.T, storage Storage, count int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var counter atomic.Int64

	var workers []*testWorker
	for range count {
		worker := &testWorker{
			election: New(ctx, storage, opts...),
			job: func() {
				counter.Add(1)
			},
		}
		workers = append(workers, worker)
	}

	time.Sleep(100 * time.Millisecond)

	t.Run("only one worker will launch", func(t *testing.T) {
		counter.Store(0)

		for _, w := range workers {
			go w.Run()
		}
		time.Sleep(30 * time.Millisecond)

		require.Equal(t, 1, int(counter.Load()))
	})

	t.Run("if we shutdown the leader worker, another worker will continue the work", func(t *testing.T) {
		counter.Store(0)

		var leader *testWorker
		for _, w := range workers {
			leader = w
			// leader now will not access the storage
			badStorage := wrap(w.election.storage)
			badStorage.shutdown(5 * time.Second)
			w.election.storage = badStorage
		}

		for _, w := range workers {
			go w.Run()
		}
		time.Sleep(30 * time.Millisecond)

		require.Equal(t, 1, int(counter.Load()))

		// doublecheck that prev leader will not run the job
		counter.Store(0)
		leader.Run()
		require.Equal(t, 0, int(counter.Load()))
	})

	t.Run("if we cancel, all workers will not run", func(t *testing.T) {
		counter.Store(0)

		cancel()
		time.Sleep(100 * time.Millisecond)

		for _, w := range workers {
			go w.Run()
		}
		time.Sleep(30 * time.Millisecond)

		require.Equal(t, 0, int(counter.Load()))
	})
}

type shutdownWrapper struct {
	s    Storage
	down atomic.Bool
}

func wrap(s Storage) *shutdownWrapper {
	return &shutdownWrapper{s: s, down: atomic.Bool{}}
}

func (wr *shutdownWrapper) shutdown(dur time.Duration) {
	go func() {
		wr.down.Store(true)
		<-time.After(dur)
		wr.down.Store(false)
	}()
}

func (wr *shutdownWrapper) Renew(ctx context.Context, key string, deadline time.Time) error {
	if wr.down.Load() {
		return errors.New("node is down")
	}

	return wr.s.Renew(ctx, key, deadline)
}

func (wr *shutdownWrapper) Get(ctx context.Context, key string) (string, error) {
	if wr.down.Load() {
		return "", errors.New("node is down")
	}

	return wr.s.Get(ctx, key)
}

func (wr *shutdownWrapper) SetNX(ctx context.Context, key string, val string, deadline time.Time) (bool, error) {
	if wr.down.Load() {
		return false, errors.New("node is down")
	}

	return wr.s.SetNX(ctx, key, val, deadline)
}

// will panic if launched 2 times at the same time
type testWorker struct {
	election *Candidate
	job      func()
}

func (w *testWorker) Run() {
	if !w.election.IsLeader() {
		return
	}

	w.job()
}

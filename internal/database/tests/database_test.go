package tests

import (
	"context"
	"errors"
	dbpkg "storage/internal/database"
	"storage/internal/database/compute"
	storagepkg "storage/internal/database/storage"
	"testing"

	"go.uber.org/zap"
)

type stubCompute struct {
	parseFn func(string) (compute.Query, error)
}

func (s stubCompute) Parse(queryStr string) (compute.Query, error) {
	return s.parseFn(queryStr)
}

type stubStorage struct {
	setFn func(context.Context, string, string) error
	getFn func(context.Context, string) (string, error)
	delFn func(context.Context, string) error
}

func (s stubStorage) Set(ctx context.Context, key, value string) error {
	return s.setFn(ctx, key, value)
}

func (s stubStorage) Get(ctx context.Context, key string) (string, error) {
	return s.getFn(ctx, key)
}

func (s stubStorage) Del(ctx context.Context, key string) error {
	return s.delFn(ctx, key)
}

func TestDB_HandleQuery_ParseError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	want := "Failed to parse query: BAD"

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) { return compute.Query{}, errors.New("boom") }},
		stubStorage{
			setFn: func(context.Context, string, string) error { return nil },
			getFn: func(context.Context, string) (string, error) { return "", nil },
			delFn: func(context.Context, string) error { return nil },
		},
	)

	got := db.HandleQuery(ctx, "BAD")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDB_HandleQuery_UnknownCommand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.UnknownCommandID, nil), nil
		}},
		stubStorage{
			setFn: func(context.Context, string, string) error { return nil },
			getFn: func(context.Context, string) (string, error) { return "", nil },
			delFn: func(context.Context, string) error { return nil },
		},
	)

	got := db.HandleQuery(ctx, "UNKNOWN")
	if got != "[error] internal error" {
		t.Fatalf("expected %q, got %q", "[error] internal error", got)
	}
}

func TestDB_HandleQuery_SetOk(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var gotKey, gotValue string

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.SetCommandID, []string{"k", "v"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, key, value string) error {
				gotKey = key
				gotValue = value
				return nil
			},
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", nil
			},
			delFn: func(_ context.Context, _ string) error {
				return nil
			},
		},
	)

	got := db.HandleQuery(ctx, "SET k v")
	if got != "[ok]" {
		t.Fatalf("expected %q, got %q", "[ok]", got)
	}
	if gotKey != "k" || gotValue != "v" {
		t.Fatalf("expected set(%q,%q), got set(%q,%q)", "k", "v", gotKey, gotValue)
	}
}

func TestDB_HandleQuery_SetError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.SetCommandID, []string{"k", "v"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, _ string, _ string) error {
				return errors.New("set failed")
			},
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", nil
			},
			delFn: func(_ context.Context, _ string) error {
				return nil
			},
		},
	)

	got := db.HandleQuery(ctx, "SET k v")
	if got != "[error] set failed" {
		t.Fatalf("expected %q, got %q", "[error] set failed", got)
	}
}

func TestDB_HandleQuery_GetFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.GetCommandID, []string{"k"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, _ string, _ string) error { return nil },
			getFn: func(_ context.Context, key string) (string, error) {
				if key != "k" {
					t.Fatalf("expected key %q, got %q", "k", key)
				}
				return "v", nil
			},
			delFn: func(_ context.Context, _ string) error { return nil },
		},
	)

	got := db.HandleQuery(ctx, "GET k")
	if got != "[ok] v" {
		t.Fatalf("expected %q, got %q", "[ok] v", got)
	}
}

func TestDB_HandleQuery_GetNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.GetCommandID, []string{"missing"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, _ string, _ string) error { return nil },
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", storagepkg.ErrorNotFound
			},
			delFn: func(_ context.Context, _ string) error { return nil },
		},
	)

	got := db.HandleQuery(ctx, "GET missing")
	if got != "[not found]" {
		t.Fatalf("expected %q, got %q", "[not found]", got)
	}
}

func TestDB_HandleQuery_GetError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.GetCommandID, []string{"k"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, _ string, _ string) error { return nil },
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("storage down")
			},
			delFn: func(_ context.Context, _ string) error { return nil },
		},
	)

	got := db.HandleQuery(ctx, "GET k")
	if got != "[error] storage down" {
		t.Fatalf("expected %q, got %q", "[error] storage down", got)
	}
}

func TestDB_HandleQuery_DelOk(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var gotKey string

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.DelCommandID, []string{"k"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, _ string, _ string) error { return nil },
			getFn: func(_ context.Context, _ string) (string, error) { return "", nil },
			delFn: func(_ context.Context, key string) error {
				gotKey = key
				return nil
			},
		},
	)

	got := db.HandleQuery(ctx, "DEL k")
	if got != "[ok]" {
		t.Fatalf("expected %q, got %q", "[ok]", got)
	}
	if gotKey != "k" {
		t.Fatalf("expected del key %q, got %q", "k", gotKey)
	}
}

func TestDB_HandleQuery_DelError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, _ := dbpkg.NewDB(
		zap.NewNop(),
		stubCompute{parseFn: func(string) (compute.Query, error) {
			return compute.NewQuery(compute.DelCommandID, []string{"k"}), nil
		}},
		stubStorage{
			setFn: func(_ context.Context, _ string, _ string) error { return nil },
			getFn: func(_ context.Context, _ string) (string, error) { return "", nil },
			delFn: func(_ context.Context, _ string) error {
				return errors.New("delete failed")
			},
		},
	)

	got := db.HandleQuery(ctx, "DEL k")
	if got != "[error] delete failed" {
		t.Fatalf("expected %q, got %q", "[error] delete failed", got)
	}
}

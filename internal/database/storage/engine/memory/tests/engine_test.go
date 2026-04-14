package tests

import (
	"context"
	"storage/internal/config"
	"storage/internal/database/storage/engine/memory"
	"testing"
)

func TestEngine_SetGetDel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	e, err := memory.NewEngine(&config.EngineConfig{Type: "in_memory"})
	if err != nil {
		t.Fatal("expected key no errors", err)
	}

	e.Set(ctx, "k", "v")

	val, ok := e.Get(ctx, "k")
	if !ok {
		t.Fatalf("expected key to exist")
	}
	if val != "v" {
		t.Fatalf("expected %q, got %q", "v", val)
	}

	e.Del(ctx, "k")

	_, ok = e.Get(ctx, "k")
	if ok {
		t.Fatalf("expected key to be deleted")
	}
}

func TestEngine_Overwrite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	e, err := memory.NewEngine(&config.EngineConfig{Type: "in_memory"})
	if err != nil {
		t.Fatal("expected key no errors", err)
	}

	e.Set(ctx, "k", "v1")
	e.Set(ctx, "k", "v2")

	val, ok := e.Get(ctx, "k")
	if !ok {
		t.Fatalf("expected key to exist")
	}
	if val != "v2" {
		t.Fatalf("expected %q, got %q", "v2", val)
	}
}

func TestEngine_GetMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	e, err := memory.NewEngine(&config.EngineConfig{Type: "in_memory"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, ok := e.Get(ctx, "missing")
	if ok {
		t.Fatalf("expected key to be missing")
	}
}

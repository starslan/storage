package tests

import (
	"errors"
	"storage/internal/database/compute"
	parserpkg "storage/internal/database/compute/parser"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestParserParse_TooLongQuery(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 10)

	long := strings.Repeat("a", 11)
	_, err := p.Parse(long)
	if !errors.Is(err, compute.ErrQueryTooLong) {
		t.Fatalf("expected ErrQueryTooLong, got %v", err)
	}
}

func TestParserParse_InvalidQuery_Empty(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 10)

	_, err := p.Parse("")
	if !errors.Is(err, compute.ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery, got %v", err)
	}
}

func TestParserParse_InvalidQuery_OnlySpaces(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 10)

	_, err := p.Parse("   \t  ")
	if !errors.Is(err, compute.ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery, got %v", err)
	}
}

func TestParserParse_InvalidCommand(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 100)

	_, err := p.Parse("FOO bar")
	if !errors.Is(err, compute.ErrInvalidCommand) {
		t.Fatalf("expected ErrInvalidCommand, got %v", err)
	}
}

func TestParserParse_InvalidArguments(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 100)

	_, err := p.Parse("SET keyOnly")
	if !errors.Is(err, compute.ErrInvalidArguments) {
		t.Fatalf("expected ErrInvalidArguments, got %v", err)
	}
}

func TestParserParse_ValidSET(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 100)

	q, err := p.Parse("SET key value")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.CommandID() != compute.SetCommandID {
		t.Fatalf("expected command id %d, got %d", compute.SetCommandID, q.CommandID())
	}
	gotArgs := q.Arguments()
	wantArgs := []string{"key", "value"}
	if len(gotArgs) != len(wantArgs) || gotArgs[0] != wantArgs[0] || gotArgs[1] != wantArgs[1] {
		t.Fatalf("expected args %v, got %v", wantArgs, gotArgs)
	}
}

func TestParserParse_ValidGET(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 100)

	q, err := p.Parse("GET mykey")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.CommandID() != compute.GetCommandID {
		t.Fatalf("expected command id %d, got %d", compute.GetCommandID, q.CommandID())
	}
	gotArgs := q.Arguments()
	if len(gotArgs) != 1 || gotArgs[0] != "mykey" {
		t.Fatalf("expected args [mykey], got %v", gotArgs)
	}
}

func TestParserParse_ValidDEL(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	p := parserpkg.NewParser(logger, 100)

	q, err := p.Parse("DEL k")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.CommandID() != compute.DelCommandID {
		t.Fatalf("expected command id %d, got %d", compute.DelCommandID, q.CommandID())
	}
	gotArgs := q.Arguments()
	if len(gotArgs) != 1 || gotArgs[0] != "k" {
		t.Fatalf("expected args [k], got %v", gotArgs)
	}
}

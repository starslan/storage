package storage

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

var (
	ErrorNotFound = errors.New("not found")
)

type Engine interface {
	Set(context.Context, string, string)
	Get(context.Context, string) (string, bool)
	Del(context.Context, string)
}

type Storage struct {
	engine Engine
	logger *zap.Logger
}

func NewStorage(engine Engine, logger *zap.Logger) (*Storage, error) {
	if engine == nil {
		return nil, errors.New("engine is invalid")
	}

	if logger == nil {
		return nil, errors.New("logger is invalid")
	}

	storage := &Storage{
		engine: engine,
		logger: logger,
	}

	return storage, nil
}

func (s *Storage) Set(ctx context.Context, key, value string) error {
	s.engine.Set(ctx, key, value)
	return nil
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	result, ok := s.engine.Get(ctx, key)
	if !ok {
		s.logger.Warn("no such key", zap.String("key", key))
		return "", ErrorNotFound
	}
	return result, nil
}

func (s *Storage) Del(ctx context.Context, key string) error {
	s.engine.Del(ctx, key)
	return nil
}

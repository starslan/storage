package memory

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

type Engine struct {
	data   *HashTable
	logger *zap.Logger
}

func NewEngine(logger *zap.Logger) (*Engine, error) {
	if logger == nil {
		return nil, errors.New("logger is invalid")
	}

	engine := &Engine{
		logger: logger,
	}

	engine.data = NewHashTable()

	return engine, nil
}
func (e *Engine) Get(_ context.Context, key string) (string, bool) {
	result, ok := e.data.Get(key)
	if !ok {
		return "", false
	}
	return result, true
}

func (e *Engine) Del(_ context.Context, key string) {
	e.data.Del(key)
}

func (e *Engine) Set(_ context.Context, key, value string) {
	e.data.Set(key, value)
}

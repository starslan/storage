package memory

import (
	"context"
	"errors"
	"storage/internal/config"
)

type Engine struct {
	data *HashTable
}

func NewEngine(config *config.EngineConfig) (*Engine, error) {
	switch config.Type {
	case "in_memory":
		return &Engine{
			data: NewHashTable(),
		}, nil
	default:
		return nil, errors.New("engine not found")
	}
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

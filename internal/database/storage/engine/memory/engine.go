package memory

import (
	"context"
)

type Engine struct {
	data *HashTable
}

func NewEngine() *Engine {

	engine := &Engine{
		data: NewHashTable(),
	}
	return engine
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

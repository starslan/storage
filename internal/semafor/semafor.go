package semafor

import (
	"storage/internal/config"
	"sync"
)

type Sem struct {
	count int
	max   int
	cond  *sync.Cond
}

func NewSem(cfg config.NetworkConfig) *Sem {
	return &Sem{
		max:  cfg.MaxConnections,
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (sem *Sem) Acquire() {
	sem.cond.L.Lock()
	defer sem.cond.L.Unlock()
	for sem.count >= sem.max {
		sem.cond.Wait()
	}
	sem.count++
}

func (sem *Sem) TryAcquire() bool {
	sem.cond.L.Lock()
	defer sem.cond.L.Unlock()
	if sem.count >= sem.max {
		return false
	}
	sem.count++
	return true
}

func (sem *Sem) Release() {
	sem.cond.L.Lock()
	defer sem.cond.L.Unlock()
	sem.count--
	sem.cond.Signal()
}

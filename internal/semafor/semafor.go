package semafor

import (
	"storage/internal/config"
	"sync"
)

type Semafor struct {
	count int
	max   int
	cond  *sync.Cond
}

func NewSemafor(cfg config.NetworkConfig) *Semafor {
	return &Semafor{
		max:  cfg.MaxConnections,
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (sem *Semafor) Acquire() {
	sem.cond.L.Lock()
	defer sem.cond.L.Unlock()
	for sem.count >= sem.max {
		sem.cond.Wait()
	}
	sem.count++
}

func (sem *Semafor) TryAcquire() bool {
	sem.cond.L.Lock()
	defer sem.cond.L.Unlock()
	if sem.count >= sem.max {
		return false
	}
	sem.count++
	return true
}

func (sem *Semafor) Release() {
	sem.cond.L.Lock()
	defer sem.cond.L.Unlock()
	sem.count--
	sem.cond.Signal()
}

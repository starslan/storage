package replication

import (
	"context"
	"storage/internal/wal"
	"time"
)

type TCPClient interface {
	Send(context.Context) error
}
type Slave struct {
	client TCPClient
	stream chan []wal.Record

	syncInterval time.Duration
	walDirectory string
}

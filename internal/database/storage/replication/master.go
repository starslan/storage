package replication

import (
	"context"

	"go.uber.org/zap"
)

type TCPServer interface {
	HandeQueries(ctx context.Context, query string) error
}
type Master struct {
	server  TCPServer
	walPath string
	logger  *zap.Logger
}

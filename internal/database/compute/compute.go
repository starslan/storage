package compute

import (
	"errors"

	"go.uber.org/zap"
)

var (
	ErrInvalidQuery     = errors.New("empty query")
	ErrInvalidCommand   = errors.New("invalid command")
	ErrInvalidArguments = errors.New("invalid arguments")
	ErrQueryTooLong     = errors.New("query exceeds maximum length")
)

type Parser interface {
	Parse(queryStr string) (Query, error)
}
type Compute struct {
	logger *zap.Logger
	parser Parser
}

func NewCompute(logger *zap.Logger, parser Parser) (*Compute, error) {
	if logger == nil {
		return nil, errors.New("logger is invalid")
	}

	if parser == nil {
		return nil, errors.New("logger is invalid")
	}
	return &Compute{
		logger: logger,
		parser: parser,
	}, nil
}

func (c *Compute) Parse(queryStr string) (Query, error) {
	return c.parser.Parse(queryStr)
}

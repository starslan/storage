package compute

import (
	"errors"
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
	parser Parser
}

func NewCompute(parser Parser) (*Compute, error) {
	if parser == nil {
		return nil, errors.New("parser is invalid")
	}
	return &Compute{
		parser: parser,
	}, nil
}

func (c *Compute) Parse(queryStr string) (Query, error) {
	return c.parser.Parse(queryStr)
}

package parser

import (
	"storage/internal/database/compute"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"
)

type Parser struct {
	logger         *zap.Logger
	maxQueryLength int
}

func NewParser(logger *zap.Logger, maxQueryLength int) *Parser {
	return &Parser{
		maxQueryLength: maxQueryLength,
		logger:         logger,
	}
}

func (p *Parser) Parse(queryStr string) (compute.Query, error) {
	if utf8.RuneCountInString(queryStr) > p.maxQueryLength {
		p.logger.Info("query exceeds maximum length",
			zap.String("query", queryStr), zap.Int("max_length", p.maxQueryLength))
		return compute.Query{}, compute.ErrQueryTooLong
	}

	tokens := strings.Fields(queryStr)
	if len(tokens) == 0 {
		p.logger.Info("request is invalid", zap.String("query", queryStr))
		return compute.Query{}, compute.ErrInvalidQuery
	}

	command := tokens[0]
	spec, ok := compute.CommandSpecList[command]
	if !ok {
		p.logger.Warn("invalid command", zap.String("query", queryStr))
		return compute.Query{}, compute.ErrInvalidCommand
	}

	query := compute.NewQuery(spec.ID, tokens[1:])
	if len(query.Arguments()) != spec.ArgCount {
		p.logger.Warn("invalid arguments for query", zap.String("query", queryStr))
		return compute.Query{}, compute.ErrInvalidArguments
	}

	return query, nil
}

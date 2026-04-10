package server

import (
	"bufio"
	"context"
	"net"
	"storage/internal/app"
	"storage/internal/config"
	"storage/internal/semafor"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	app      *app.App
	listener net.Listener
	cfg      config.NetworkConfig
	logger   *zap.Logger
	sem      *semafor.Sem
}

func NewServer(app *app.App, cfg config.NetworkConfig) (*Server, error) {
	ln, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, err
	}

	return &Server{
		app:      app,
		listener: ln,
		cfg:      cfg,
		logger:   app.Logger,
		sem:      semafor.NewSem(cfg),
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Server listen:", zap.String("addr", s.listener.Addr().String()))

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.logger.Error("Server accept error", zap.Error(err))
		}

		if !s.sem.TryAcquire() {
			s.logger.Warn("Connection limit exceeded")

			_, _ = conn.Write([]byte("Too many connections, try again later\n"))
			_ = conn.Close()
			continue
		}

		timeout, err := time.ParseDuration(s.cfg.IdleTimeout)
		if err == nil {
			err = conn.SetDeadline(time.Now().Add(timeout))
			if err != nil {
				s.logger.Warn("Failed to set connection deadline", zap.Error(err))
			}
		}
		connCtx, cancel := context.WithCancel(ctx)
		go func() {
			defer cancel()
			defer s.sem.Release()
			s.handleConnection(connCtx, conn)
		}()
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	s.logger.Info("Client connected:", zap.String("remote addr", conn.RemoteAddr().String()))
	maxPageSize := ParseSize(s.cfg.MaxMessageSize)
	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.logger.Info("Client disconnected:", zap.Error(err))
			return
		}

		if len(msg) > maxPageSize {
			s.logger.Warn("Message too large")
			_, _ = conn.Write([]byte("Message too large\n"))
			return
		}

		result := s.app.DB.HandleQuery(ctx, msg)

		_, err = conn.Write([]byte(result + "\n"))
		if err != nil {
			s.logger.Error("Write result error:", zap.Error(err))
			return
		}
	}
}

func (s *Server) Stop() error {
	return s.listener.Close()
}

func ParseSize(s string) int {
	s = strings.ToUpper(strings.TrimSpace(s))

	multiplier := 1
	switch {
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	}

	n, _ := strconv.Atoi(s)
	return n * multiplier
}

package mctcp

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"net"
)

type Server struct {
	ctx        context.Context
	size       int
	listenAddr string

	pool *TcpPool
}

func NewServer(ctx context.Context, size int, addr string) (*Server, error) {
	s := new(Server)
	s.size = size
	s.listenAddr = addr
	s.ctx = ctx
	s.pool = NewPool(ctx, size)
	go func() {
		err := s.ListenAndAccept()
		if err != nil {
			zap.L().Error("listen and accept", zap.Error(err))
		}
	}()
	return s, nil
}

func (s *Server) ListenAndAccept() error {
	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		zap.L().Debug("accept tcp connection", zap.String("from", conn.RemoteAddr().String()))
		s.pool.Push(conn.(*net.TCPConn))
	}
}

func (s *Server) Read() ([]byte, error) {
	return s.pool.Read()
}

func (s *Server) Write(buf []byte) error {
	err := s.pool.Write(buf)
	if err != nil && !errors.Is(err, ErrBrokenConn) {
		return err
	}
	return nil
}

package mctcp

import (
	"context"
	"log"
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
			log.Printf("listen and accept: %v", err)
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
		s.pool.Push(conn.(*net.TCPConn))
	}
}

func (s *Server) Read() ([]byte, error) {
	return s.pool.Read()
}

func (s *Server) Write(buf []byte) error {
	return s.pool.Write(buf)
}

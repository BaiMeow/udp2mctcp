package mctcp

import (
	"net"
)

type MarkConn struct {
	*net.TCPConn
	available bool
}

func NewMarkConn(conn *net.TCPConn) *MarkConn {
	return &MarkConn{conn, true}
}

func (m *MarkConn) Close() error {
	m.available = false
	return m.TCPConn.Close()
}

func (m *MarkConn) IsAvailable() bool {
	return m.available
}

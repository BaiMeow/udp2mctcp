package mctcp

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
)

type TcpPool struct {
	ctx context.Context

	current atomic.Int32
	size    atomic.Int32

	// lock for writerPool
	lock       sync.RWMutex
	writerPool chan *net.TCPConn

	readBuffer         chan []byte
	eventTcpReadFailed func(err error)
}

func NewPool(ctx context.Context, size int) *TcpPool {
	p := &TcpPool{
		writerPool: make(chan *net.TCPConn, size),
		readBuffer: make(chan []byte, ReadBufferSize),
		ctx:        ctx,
	}
	p.size.Store(int32(size))
	p.current.Store(0)
	return p
}

func (p *TcpPool) watchTcp(c *net.TCPConn) {
	for {
		if p.Closed() {
			_ = c.Close()
			return
		}
		buf, err := Stream2Packet(c)
		if err != nil {
			if p.eventTcpReadFailed != nil {
				p.eventTcpReadFailed(err)
			}
			return
		}
		select {
		case p.readBuffer <- buf:
		default:
			// drop packet
		}
	}
}

func (p *TcpPool) Closed() bool {
	select {
	case <-p.ctx.Done():
		return true
	default:
		return false
	}
}

func (p *TcpPool) Push(conn *net.TCPConn) {
	p.current.Add(1)
RETRY:
	oldCurrent := p.current.Load()
	oldSize := p.size.Load()
	if oldCurrent > oldSize {
		if !p.size.CompareAndSwap(oldSize, oldSize*2) {
			goto RETRY
		}
		// enlarge
		p.lock.Lock()
		oldPool := p.writerPool
		p.writerPool = make(chan *net.TCPConn, oldSize*2)
		close(oldPool)
		for c := range oldPool {
			p.writerPool <- c
		}
		p.lock.Unlock()
	}

	p.writerPool <- conn
	go p.watchTcp(conn)
}

func (p *TcpPool) RegisterTcpReadFailed(fn func(err error)) {
	p.eventTcpReadFailed = fn
}

func (p *TcpPool) Read() ([]byte, error) {
	select {
	case <-p.ctx.Done():
		return nil, ErrClosed
	case buf := <-p.readBuffer:
		return buf, nil
	}
}

func (p *TcpPool) Write(buf []byte) error {
	if p.Closed() {
		return ErrClosed
	}
	p.lock.RLock()
	select {
	case conn := <-p.writerPool:
		p.lock.RUnlock()
		if err := Packet2Stream(buf, conn); err != nil {
			return err
		}
		p.lock.RLock()
		p.writerPool <- conn
		p.lock.RUnlock()
	default:
		// drop packet
		p.lock.RUnlock()
	}
	return nil
}

package mctcp

import (
	"bufio"
	"context"
	"go.uber.org/zap"
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
	writerPool chan *MarkConn

	readBuffer         chan []byte
	eventTcpReadFailed func(err error)
}

func NewPool(ctx context.Context, size int, readBufferSize int) *TcpPool {
	p := &TcpPool{
		writerPool: make(chan *MarkConn, size),
		readBuffer: make(chan []byte, readBufferSize),
		ctx:        ctx,
	}
	p.size.Store(int32(size))
	p.current.Store(0)
	return p
}

func (p *TcpPool) watchTcp(c *MarkConn) {
	r := bufio.NewReader(c)
	defer c.Close()
	for {
		if p.Closed() {
			return
		}
		buf, err := Stream2Packet(r)
		if err != nil {
			zap.L().Debug("read tcp", zap.Error(err))
			if p.eventTcpReadFailed != nil {
				p.eventTcpReadFailed(err)
			}
			return
		}
		zap.L().Debug("tcp->",
			zap.Int("len", len(buf)),
			zap.String("from", c.RemoteAddr().String()),
			zap.String("to", c.LocalAddr().String()))
		select {
		case p.readBuffer <- buf:
		default:
			zap.L().Debug("read buffer full, drop packet")
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
	cur := p.current.Add(1)
	zap.L().Debug("add connection", zap.Int("current", int(cur)))
RETRY:
	oldCurrent := p.current.Load()
	oldSize := p.size.Load()
	if oldCurrent > oldSize {
		if !p.size.CompareAndSwap(oldSize, oldSize*2) {
			goto RETRY
		}
		// enlarge
		zap.L().Debug("enlarge writer pool", zap.Int("newsize", int(oldSize*2)))
		p.lock.Lock()
		oldPool := p.writerPool
		p.writerPool = make(chan *MarkConn, oldSize*2)
		close(oldPool)
		for c := range oldPool {
			p.writerPool <- c
		}
		p.lock.Unlock()
	}
	p.lock.RLock()
	markConn := NewMarkConn(conn)
	p.writerPool <- markConn
	p.lock.RUnlock()
	go p.watchTcp(markConn)
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
RETRY:
	select {
	case conn := <-p.writerPool:
		if !conn.IsAvailable() {
			goto RETRY
		}
		p.lock.RUnlock()
		if err := Packet2Stream(buf, conn); err != nil {
			p.current.Add(-1)
			return err
		}
		p.lock.RLock()
		zap.L().Debug("->tcp", zap.Int("len", len(buf)))
		p.writerPool <- conn
		p.lock.RUnlock()
	default:
		// drop packet
		p.lock.RUnlock()
	}
	return nil
}

package mctcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/BaiMeow/udp2mctcp/utils"
	"net"
	"sync"
)

var (
	ErrClosed     = errors.New("pool closed")
	ErrBrokenConn = errors.New("broken connection")
)

const ReadBufferSize = 128

type Client struct {
	ctx      context.Context
	size     int
	dialAddr string

	resumeCounter      int
	readBrokenCounter  int
	writeBrokenCounter int
	counterLock        sync.Mutex

	pool *TcpPool
}

func NewClient(ctx context.Context, size int, addr string) (*Client, error) {
	c := new(Client)
	c.size = size
	c.dialAddr = addr
	c.ctx = ctx
	c.pool = NewPool(ctx, size)
	c.pool.RegisterTcpReadFailed(func(err error) {
		if errors.Is(err, ErrBrokenConn) {
			c.counterLock.Lock()
			defer c.counterLock.Unlock()
			c.readBrokenCounter++
			if c.readBrokenCounter > c.resumeCounter {
				c.resumeCounter++
			}
			go func() {
				_ = utils.Retry(func() error {
					return c.addConn()
				}, 3)
			}()
		}

	})
	for range size {
		tconn, err := createConnection(addr)
		if err != nil {
			return nil, fmt.Errorf("create connection failed: %v", err)
		}
		c.pool.Push(tconn)
	}
	return c, nil
}

// Write select an available conn and then write data to it
func (c *Client) Write(buf []byte) error {
	if c.Closed() {
		return ErrClosed
	}

	err := c.pool.Write(buf)
	if errors.Is(err, ErrBrokenConn) {
		c.counterLock.Lock()
		defer c.counterLock.Unlock()
		c.writeBrokenCounter++
		if c.writeBrokenCounter > c.resumeCounter {
			c.resumeCounter++
		}
		go func() {
			_ = utils.Retry(func() error {
				return c.addConn()
			}, 3)
		}()
		return nil
	}
	return err
}

func (c *Client) Read() ([]byte, error) {
	if c.Closed() {
		return nil, ErrClosed
	}
	return c.pool.Read()
}

func (c *Client) Closed() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

func (c *Client) addConn() error {
	tconn, err := createConnection(c.dialAddr)
	if err != nil {
		return fmt.Errorf("create connection failed: %v", err)
	}
	c.pool.Push(tconn)
	return nil
}

func createConnection(addr string) (*net.TCPConn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	tconn := conn.(*net.TCPConn)
	if err := tconn.SetKeepAlive(true); err != nil {
		return nil, fmt.Errorf("enable keepalive failed: %v", err)
	}
	if err := tconn.SetNoDelay(true); err != nil {
		return nil, fmt.Errorf("set nodelay failed: %v", err)
	}
	return conn.(*net.TCPConn), nil
}

package main

import (
	"context"
	"flag"
	"github.com/BaiMeow/udp2mctcp/forward"
	"github.com/BaiMeow/udp2mctcp/log"
	"github.com/BaiMeow/udp2mctcp/mctcp"
	"go.uber.org/zap"
	"net"
)

const DefaultPoolSize = 16

var (
	forwarderAddr string
	tcpListenAddr string
)

func main() {
	flag.StringVar(&forwarderAddr, "f", "", "udp forwarder addr")
	flag.StringVar(&tcpListenAddr, "l", "", "tcp listen addr")
	level := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()
	log.Init(*level)

	conn, err := net.Dial("udp", forwarderAddr)
	if err != nil {
		zap.L().Fatal("dial udp", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	s, err := mctcp.NewServer(ctx, DefaultPoolSize, tcpListenAddr)
	if err != nil {
		zap.L().Fatal("new server", zap.Error(err))
	}

	// mctcp -> udp
	go func() {
		err := forward.Mctcp2Udp(s, conn.(*net.UDPConn))
		zap.L().Fatal("mctcp2udp fail", zap.Error(err))
		cancel()
	}()

	// udp -> mctcp
	go func() {
		err := forward.Udp2Mctcp(conn.(*net.UDPConn), s)
		zap.L().Fatal("udp2mctcp fail", zap.Error(err))
		cancel()
	}()

	<-ctx.Done()
}

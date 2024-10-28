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

var (
	udpListenAddr string
	forwarderAddr string
	connCount     int
)

func main() {
	flag.StringVar(&udpListenAddr, "l", "", "udp listen port")
	flag.StringVar(&forwarderAddr, "f", "", "tcp forwarder addr")
	flag.IntVar(&connCount, "c", 8, "tcp connection count")
	level := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()
	log.Init(*level)

	conn, err := net.ListenPacket("udp", udpListenAddr)
	if err != nil {
		zap.L().Fatal("listen udp", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	c, err := mctcp.NewClient(ctx, connCount, forwarderAddr)
	if err != nil {
		zap.L().Fatal("new client", zap.Error(err))
	}

	// mctcp -> udp
	go func() {
		err := forward.Mctcp2Udp(c, conn.(*net.UDPConn))
		zap.L().Fatal("mctcp2udp fail", zap.Error(err))
		cancel()
	}()

	// udp -> mctcp
	go func() {
		err := forward.Udp2Mctcp(conn.(*net.UDPConn), c)
		zap.L().Fatal("udp2mctcp fail", zap.Error(err))
		cancel()
	}()

	<-ctx.Done()
}

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
	udpForwarderAddr string
	tcpListenAddr    string
	readBufferSize   int
)

func main() {
	flag.StringVar(&udpForwarderAddr, "f", "", "udp forwarder addr")
	flag.StringVar(&tcpListenAddr, "l", "", "tcp listen addr")
	flag.IntVar(&readBufferSize, "b", 4096, "read buffer size")
	level := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()
	log.Init(*level)

	conn, err := net.Dial("udp", udpForwarderAddr)
	if err != nil {
		zap.L().Fatal("dial udp", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	s, err := mctcp.NewServer(ctx, DefaultPoolSize, readBufferSize, tcpListenAddr)
	if err != nil {
		zap.L().Fatal("new server", zap.Error(err))
	}

	// mctcp -> udp
	go func() {
		err := forward.Mctcp2Udp(s, conn.(*net.UDPConn))
		zap.L().Fatal("mctcp2udp fail", zap.Error(err))
		cancel()
	}()
	zap.L().Info("run mctcp -> udp")

	// udp -> mctcp
	go func() {
		err := forward.Udp2Mctcp(conn.(*net.UDPConn), s)
		zap.L().Fatal("udp2mctcp fail", zap.Error(err))
		cancel()
	}()
	zap.L().Info("run udp -> mctcp")

	zap.L().Info("setup done")
	<-ctx.Done()
}

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
	udpListenAddr  string
	forwarderAddr  string
	connCount      int
	readBufferSize int
)

func main() {
	flag.StringVar(&udpListenAddr, "l", "", "udp listen port")
	flag.StringVar(&forwarderAddr, "f", "", "tcp forwarder addr")
	flag.IntVar(&connCount, "c", 8, "tcp connection count")
	flag.IntVar(&readBufferSize, "b", 4096, "read buffer size")
	level := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()
	log.Init(*level)

	conn, err := net.ListenPacket("udp", udpListenAddr)
	if err != nil {
		zap.L().Fatal("listen udp", zap.Error(err))
	}

	// drop first packet, and fetch the addr
	zap.L().Info("waiting for first udp packet, to get udp forward addr")
	_, addr, err := conn.ReadFrom(nil)
	if err != nil {
		zap.L().Fatal("read from", zap.Error(err))
	}
	zap.L().Info("got udp forward addr", zap.String("addr", addr.String()))

	if err := conn.Close(); err != nil {
		zap.L().Fatal("close udp", zap.Error(err))
	}

	conn, err = net.DialUDP("udp", conn.LocalAddr().(*net.UDPAddr), addr.(*net.UDPAddr))
	if err != nil {
		zap.L().Fatal("dial udp", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	c, err := mctcp.NewClient(ctx, connCount, readBufferSize, forwarderAddr)
	if err != nil {
		zap.L().Fatal("new client", zap.Error(err))
	}

	// mctcp -> udp
	go func() {
		err := forward.Mctcp2Udp(c, conn.(*net.UDPConn))
		zap.L().Fatal("mctcp2udp fail", zap.Error(err))
		cancel()
	}()
	zap.L().Info("run mctcp -> udp")

	// udp -> mctcp
	go func() {
		err := forward.Udp2Mctcp(conn.(*net.UDPConn), c)
		zap.L().Fatal("udp2mctcp fail", zap.Error(err))
		cancel()
	}()
	zap.L().Info("run udp -> mctcp")

	zap.L().Info("setup done")
	<-ctx.Done()
}

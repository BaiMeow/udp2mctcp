package main

import (
	"context"
	"flag"
	"github.com/BaiMeow/udp2mctcp/forward"
	"github.com/BaiMeow/udp2mctcp/mctcp"
	"log"
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
	flag.Parse()

	conn, err := net.Dial("udp", forwarderAddr)
	if err != nil {
		log.Fatalln(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s, err := mctcp.NewServer(ctx, DefaultPoolSize, tcpListenAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// mctcp -> udp
	go func() {
		err := forward.Mctcp2Udp(s, conn.(*net.UDPConn))
		log.Println("mctcp2udp fail:", err)
		cancel()
	}()

	// udp -> mctcp
	go func() {
		err := forward.Udp2Mctcp(conn.(*net.UDPConn), s)
		log.Println("udp2mctcp fail:", err)
		cancel()
	}()

	<-ctx.Done()
}

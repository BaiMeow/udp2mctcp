package main

import (
	"context"
	"flag"
	"github.com/BaiMeow/udp2mctcp/forward"
	"github.com/BaiMeow/udp2mctcp/mctcp"
	"log"
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
	flag.Parse()

	conn, err := net.ListenPacket("udp", udpListenAddr)
	if err != nil {
		log.Fatalln(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c, err := mctcp.NewClient(ctx, connCount, forwarderAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// mctcp -> udp
	go func() {
		err := forward.Mctcp2Udp(c, conn.(*net.UDPConn))
		log.Println("mctcp2udp fail:", err)
		cancel()
	}()

	// udp -> mctcp
	go func() {
		err := forward.Udp2Mctcp(conn.(*net.UDPConn), c)
		log.Println("udp2mctcp fail:", err)
		cancel()
	}()

	for {
		// Ethernet MTU 1500, 1600 is enough
		buf := make([]byte, 1600)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("read fail: %v", err)
			break
		}
		buf = buf[:n]
		go func() {
			err := c.Write(buf)
			if err != nil {
				log.Printf("write fail: %v", err)
			}
		}()
	}

}

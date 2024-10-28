package forward

import (
	"github.com/BaiMeow/udp2mctcp/mctcp"
	"net"
)

func Mctcp2Udp(r mctcp.Reader, conn *net.UDPConn) error {
	for {
		buf, err := r.Read()
		if err != nil {
			return err
		}
		_, err = conn.Write(buf)
		if err != nil {
			return err
		}
	}
}

func Udp2Mctcp(conn *net.UDPConn, w mctcp.Writer) error {
	writeErr := make(chan error, 1)
	for {
		buf := make([]byte, 1600)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		buf = buf[:n]

		// check writer ok
		select {
		case err := <-writeErr:
			return err
		default:
		}

		go func() {
			err := w.Write(buf[:n])
			if err != nil {
				writeErr <- err
			}
		}()
	}
}

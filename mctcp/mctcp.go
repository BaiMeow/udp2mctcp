package mctcp

type Reader interface {
	Read() ([]byte, error)
}

type Writer interface {
	Write(buf []byte) error
}

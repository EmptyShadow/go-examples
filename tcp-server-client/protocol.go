package tcp_server_client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

type Protocol struct {
	conn          net.Conn
	bytesOfNumber [10]byte
}

func NewProtocol(conn net.Conn) *Protocol {
	return &Protocol{
		conn: conn,
	}
}

func (p *Protocol) ReadNumber() (int64, error) {
	_, err := p.conn.Read(p.bytesOfNumber[:])
	if err != nil {
		return 0, fmt.Errorf("read bytes from connection: %w", err)
	}

	number, n := binary.Varint(p.bytesOfNumber[:])
	if n < 0 {
		return 0, errors.New("read large number")
	}
	if n == 0 {
		return 0, errors.New("buf is small")
	}

	return number, nil
}

func (p *Protocol) WriteNumber(number int64) error {
	binary.PutVarint(p.bytesOfNumber[:], number)

	if _, err := p.conn.Write(p.bytesOfNumber[:]); err != nil {
		return fmt.Errorf("write bytes to connection: %w", err)
	}

	return nil
}

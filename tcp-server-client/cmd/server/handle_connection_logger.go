package main

import (
	"log"
	"net"
)

type HandleConnectionLogger struct {
	logger *log.Logger
	next   HandleConnection
}

func NewHandleConnectionLogger(logger *log.Logger, next HandleConnection) *HandleConnectionLogger {
	return &HandleConnectionLogger{
		logger: logger,
		next:   next,
	}
}

func (l *HandleConnectionLogger) HandleConnection(conn net.Conn) error {
	l.logger.Println("INFO", "new connection", "local address", conn.LocalAddr(), "remote address", conn.RemoteAddr())
	defer l.logger.Println("INFO", "closed connection", "local address", conn.LocalAddr(), "remote address", conn.RemoteAddr())

	err := l.next(conn)
	if err != nil {
		log.Println("ERROR", err.Error())
	}
	return err
}

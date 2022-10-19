package main

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

type HandleConnection func(conn net.Conn) error

type ConnectionAcceptWorker struct {
	listener            net.Listener
	handleConn          HandleConnection
	groupOfConnHandlers sync.WaitGroup

	shutdownStarted int32
}

func NewConnectionAcceptWorker(netListener net.Listener, h HandleConnection) *ConnectionAcceptWorker {
	server := ConnectionAcceptWorker{
		listener:   netListener,
		handleConn: h,
	}

	return &server
}

func (s *ConnectionAcceptWorker) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("accept conn from net listener: %w", err)
		}

		if atomic.LoadInt32(&s.shutdownStarted) > 0 {
			conn.Close()
			continue
		}

		s.groupOfConnHandlers.Add(1)
		go func(conn net.Conn) {
			s.handleConn(conn)
			conn.Close()
			s.groupOfConnHandlers.Done()
		}(conn)
	}
}

func (s *ConnectionAcceptWorker) Shutdown() error {
	atomic.AddInt32(&s.shutdownStarted, 1)
	s.groupOfConnHandlers.Wait()

	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("close net listener: %w", err)
	}

	return nil
}

package tcp_server_client

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

type HandleConnection func(conn net.Conn) error

type Server struct {
	listener            net.Listener
	handleConn          HandleConnection
	groupOfConnHandlers sync.WaitGroup

	shutdownStarted int32
}

func NewServer(netListener net.Listener, h HandleConnection) *Server {
	server := Server{
		listener:   netListener,
		handleConn: h,
	}

	return &server
}

func (s *Server) Serve() error {
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

func (s *Server) Shutdown() error {
	atomic.AddInt32(&s.shutdownStarted, 1)
	s.groupOfConnHandlers.Wait()

	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("close net listener: %w", err)
	}

	return nil
}

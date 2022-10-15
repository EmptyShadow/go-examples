package tcp_server_client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
)

type Server struct {
	listener net.Listener
}

func NewServer(listener net.Listener) *Server {
	return &Server{
		listener: listener,
	}
}

func (s *Server) Serve() error {
	log.Println("start accept conn from net listener")

	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Println("stop accept conn from net listener")
			return nil
		}
		if err != nil {
			return fmt.Errorf("accept conn from net listener: %w", err)
		}

		log.Println("accept conn,", "remote address:", conn.RemoteAddr().String())
		conn.Close()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("close net listener: %w", err)
	}

	return nil
}

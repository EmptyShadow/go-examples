package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	tcp_server_client "github.com/EmptyShadow/go-examples/tcp-server-client"
)

type ConnectionHandler struct {
	numbersHandler *NumbersHandler

	handleNumberTimeout time.Duration
}

func NewConnectionHandler(numbersHandler *NumbersHandler) *ConnectionHandler {
	return &ConnectionHandler{
		numbersHandler:      numbersHandler,
		handleNumberTimeout: time.Second,
	}
}

func (h *ConnectionHandler) HandleConnection(conn net.Conn) error {
	ctx := context.TODO()
	protocol := tcp_server_client.NewProtocol(conn)

	for {
		number, err := protocol.ReadNumber()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read number: %w", err)
		}

		sumOfSquares, err := h.handleNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("handle number: %w", err)
		}

		if err = protocol.WriteNumber(sumOfSquares); err != nil {
			return fmt.Errorf("write sum of squares: %w", err)
		}
	}

	return nil
}

func (h *ConnectionHandler) handleNumber(ctx context.Context, number int64) (sumOfSquares int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, h.handleNumberTimeout)
	defer cancel()

	return h.numbersHandler.Handle(ctx, number)
}

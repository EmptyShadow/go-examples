package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
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
	buf := make([]byte, 10)

	ctx := context.Background()

	for {
		_, err := conn.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read bytes from conn: %w", err)
		}

		number, n := binary.Varint(buf)
		if n < 0 {
			return errors.New("read large number")
		}
		if n == 0 {
			return errors.New("buf is small")
		}

		sumOfSquares, err := h.handleNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("handle number: %w", err)
		}

		binary.PutVarint(buf, sumOfSquares)

		if _, err = conn.Write(buf); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}

	return nil
}

func (h *ConnectionHandler) handleNumber(ctx context.Context, number int64) (sumOfSquares int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, h.handleNumberTimeout)
	defer cancel()

	return h.numbersHandler.Handle(ctx, number)
}

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	tcp_server_client "github.com/EmptyShadow/go-examples/tcp-server-client"
)

func main() {
	log.Println("start server")

	listenAddress := flag.String("listen-address", ":9999", "address of tcp listen")
	shutdownTimeout := flag.Duration("shutdown-timeout", time.Second*30, "timeout of shutdown program")
	flag.Parse()

	handleConn := logHandleConnection(handleConnection())

	netAddress, err := net.ResolveTCPAddr("tcp", *listenAddress)
	if err != nil {
		err = fmt.Errorf("resolve tcp address by %s: %w", *listenAddress, err)
		log.Fatalln(err)
	}

	netListener, err := net.ListenTCP("tcp", netAddress)
	if err != nil {
		err = fmt.Errorf("create net listener on tcp://%s: %w", netAddress, err)
		log.Fatalln(err)
	}

	server := tcp_server_client.NewServer(netListener, handleConn)

	serveContext := context.Background()
	serveContext, serveCancel := signal.NotifyContext(serveContext, os.Interrupt)
	defer serveCancel()

	serveErr := make(chan error)
	go func() {
		err := server.Serve()
		if err == nil {
			close(serveErr)
			return
		}

		serveErr <- err
		close(serveErr)
	}()

	select {
	case <-serveContext.Done():
	case err := <-serveErr:
		err = fmt.Errorf("serve tcp server: %w", err)
		log.Fatalln(err)
	}

	shutdownContext := context.Background()
	shutdownContext, shutdownCancel := context.WithTimeout(shutdownContext, *shutdownTimeout)
	defer shutdownCancel()

	shutdownErr := make(chan error)
	go func() {
		err := server.Shutdown()
		if err == nil {
			close(shutdownErr)
			return
		}

		shutdownErr <- err
		close(shutdownErr)
	}()

	select {
	case <-shutdownContext.Done():
	case err := <-shutdownErr:
		if err == nil {
			break
		}

		err = fmt.Errorf("shutdown tcp server: %w", err)
		log.Fatalln(err)
	}

	log.Println("stop server")
}

func logHandleConnection(next tcp_server_client.HandleConnection) tcp_server_client.HandleConnection {
	return func(conn net.Conn) error {
		log.Println("connected to", conn.LocalAddr(), "from", conn.RemoteAddr())
		err := next(conn)
		if err != nil {
			log.Println("ERROR", err.Error())
		}
		return err
	}
}

func handleConnection() tcp_server_client.HandleConnection {
	return func(conn net.Conn) error {
		var sumOfSquares int64

		buf := make([]byte, 10)

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

			sumOfSquares += number * number
			log.Println("State:", sumOfSquares, "Number:", number)

			binary.PutVarint(buf, sumOfSquares)

			if _, err = conn.Write(buf); err != nil {
				return fmt.Errorf("write response: %w", err)
			}
		}

		return nil
	}
}

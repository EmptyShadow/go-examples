package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

func main() {

	listenAddress := flag.String("listen-address", ":9999", "address of tcp listen")
	shutdownTimeout := flag.Duration("shutdown-timeout", time.Second*30, "timeout of shutdown program")
	flag.Parse()

	logger := log.New(os.Stdout, "tcp-server-example -> ", log.LstdFlags)
	logger.Println("start server")

	inmemorySetOfNumbers := NewInmemorySetOfNumbers()
	numbersHandler := NewNumbersHandler(inmemorySetOfNumbers)
	connectionHandler := NewConnectionHandler(numbersHandler)
	handleConnectionLogger := NewHandleConnectionLogger(logger, connectionHandler.HandleConnection)

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

	server := NewServer(netListener, handleConnectionLogger.HandleConnection)

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

	logger.Println("stop server")
}

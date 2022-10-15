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

	tcp_server_client "github.com/EmptyShadow/go-examples/tcp-server-client"
)

func main() {
	log.Println("start server")

	listenAddress := flag.String("listen-address", ":9999", "address of tcp listen")
	shutdownTimeout := flag.Duration("shutdown-timeout", time.Second*30, "timeout of shutdown program")
	flag.Parse()

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
	defer netListener.Close()
	log.Println("listen tcp address", netListener.Addr().String())

	server := tcp_server_client.NewServer(netListener)

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
		err := server.Shutdown(shutdownContext)
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

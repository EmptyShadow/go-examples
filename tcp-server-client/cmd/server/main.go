package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

func main() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		err = fmt.Errorf("get user home dir from os: %w", err)
		log.Fatalln(err)
	}

	listenAddress := flag.String("listen-address", ":9999", "address of tcp listen")
	shutdownTimeout := flag.Duration("shutdown-timeout", time.Second*30, "timeout of shutdown program")
	dumpDir := flag.String("dump-dir", userHomeDir, "path to dump dir")
	dumpFileName := flag.String("dump-file-name", "numbers.dump", "file name of numbers dump")
	dumpPeriod := flag.Duration("dump-period", time.Second, "dump period")
	flag.Parse()

	logger := log.New(os.Stdout, "tcp-server-example -> ", log.LstdFlags)
	logger.Println("start server")

	inmemorySetOfNumbers := NewInmemorySetOfNumbers()
	dumpWorker := NewDumpWorker(inmemorySetOfNumbers, *dumpDir, *dumpFileName, *dumpPeriod)
	numbersHandler := NewNumbersHandler(inmemorySetOfNumbers)
	connectionHandler := NewConnectionHandler(numbersHandler)
	handleConnectionLogger := NewHandleConnectionLogger(logger, connectionHandler.HandleConnection)

	netAddress, err := net.ResolveTCPAddr("tcp", *listenAddress)
	if err != nil {
		err = fmt.Errorf("resolve tcp address by %s: %w", *listenAddress, err)
		logger.Fatalln(err)
	}

	netListener, err := net.ListenTCP("tcp", netAddress)
	if err != nil {
		err = fmt.Errorf("create net listener on tcp://%s: %w", netAddress, err)
		logger.Fatalln(err)
	}

	server := NewServer(netListener, handleConnectionLogger.HandleConnection)

	serveContext := context.Background()
	serveContext, cancelSignalNotify := signal.NotifyContext(serveContext, os.Interrupt)
	defer cancelSignalNotify()

	serveErrs := goFuncs(server.Serve, dumpWorker.Serve)

	select {
	case <-serveContext.Done():
		logger.Println("serve context done")
	case err := <-serveErrs:
		logger.Println("first error from serve func:", err)
	}

	shutdownContext := context.Background()
	shutdownContext, cancelWaitTimeout := context.WithTimeout(shutdownContext, *shutdownTimeout)
	defer cancelWaitTimeout()

	shutdownErrs := goFuncs(server.Shutdown, dumpWorker.Shutdown)

	var stopWait bool
	for !stopWait {
		select {
		case <-shutdownContext.Done():
			logger.Println("shutdown context done")
			stopWait = true
		case err, isOpen := <-serveErrs:
			stopWait = !isOpen
			logger.Println("error from serve func:", err)
		case err, isOpen := <-shutdownErrs:
			stopWait = !isOpen
			logger.Println("error from shutdown func:", err)
		}
	}

	logger.Println("stop server")
}

func goFuncs(fs ...func() error) <-chan error {
	count := len(fs)
	errs := make(chan error, count)
	counter := int64(count)
	fmt.Println(counter)
	for i := range fs {
		f := fs[i]
		go func() {
			errs <- f()
			n := atomic.AddInt64(&counter, -1)
			fmt.Println(atomic.LoadInt64(&counter))
			if n == 0 {
				close(errs)
			}
		}()
	}
	return errs
}

// func waitCtxOrCloseChans()

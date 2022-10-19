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
	defer logger.Println("stop server")

	netListener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		err = fmt.Errorf("create net listener on tcp://%s: %w", *listenAddress, err)
		logger.Fatalln(err)
	}
	defer netListener.Close()

	inmemorySetOfNumbers := NewInmemorySetOfNumbers()
	dumpFile := NewDumpFile(*dumpDir, *dumpFileName)
	numbersDumpWorker := NewNumbersDumpWorker(inmemorySetOfNumbers, dumpFile.Save, *dumpPeriod)
	numbersHandler := NewNumbersHandler(inmemorySetOfNumbers)
	connectionHandler := NewConnectionHandler(numbersHandler)
	handleConnectionLogger := NewHandleConnectionLogger(logger, connectionHandler.HandleConnection)
	connectionAcceptWorker := NewConnectionAcceptWorker(netListener, handleConnectionLogger.HandleConnection)

	startWork(numbersDumpWorker, connectionAcceptWorker, logger, *shutdownTimeout)
}

func startWork(
	numbersDumpWorker *NumbersDumpWorker,
	connectionAcceptWorker *ConnectionAcceptWorker,
	logger *log.Logger,
	shutdownTimeout time.Duration,
) {
	serveContext := context.Background()
	serveContext, cancelSignalNotify := signal.NotifyContext(serveContext, os.Interrupt)
	defer cancelSignalNotify()

	serveErrs := goFuncs(connectionAcceptWorker.Serve, numbersDumpWorker.Serve)

	select {
	case <-serveContext.Done():
		logger.Println("serve context done")
	case err := <-serveErrs:
		logger.Println("first error from serve func:", err)
	}

	shutdownContext := context.Background()
	shutdownContext, cancelWaitTimeout := context.WithTimeout(shutdownContext, shutdownTimeout)
	defer cancelWaitTimeout()

	shutdownErrs := goFuncs(connectionAcceptWorker.Shutdown, numbersDumpWorker.Shutdown)

	var (
		shutdownCanceled  bool
		serveCompleted    bool
		shutdownCompleted bool
	)

	for !shutdownCanceled && !serveCompleted {
		select {
		case <-shutdownContext.Done():
			logger.Println("shutdown context done")
			shutdownCanceled = true
		case err, isOpen := <-serveErrs:
			serveCompleted = !isOpen
			if err == nil {
				break
			}
			logger.Println("error from serve func:", err)
		}
	}

	for !shutdownCanceled && !shutdownCompleted {
		select {
		case <-shutdownContext.Done():
			logger.Println("shutdown context done")
			shutdownCanceled = true
		case err, isOpen := <-shutdownErrs:
			shutdownCompleted = !isOpen
			if err == nil {
				break
			}
			logger.Println("error from shutdown func:", err)
		}
	}
}

func goFuncs(fs ...func() error) <-chan error {
	count := len(fs)
	errs := make(chan error, count)
	counter := int64(count)
	for i := range fs {
		f := fs[i]
		go func() {
			errs <- f()
			if atomic.AddInt64(&counter, -1) == 0 {
				close(errs)
			}
		}()
	}
	return errs
}

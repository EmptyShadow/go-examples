package main

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

type DumpWorker struct {
	setOfNumbers   SetOfNumbers
	filePath       string
	ticker         *time.Ticker
	serveStopped   chan struct{}
	shutdownRunned chan struct{}
}

func NewDumpWorker(setOfNumbers SetOfNumbers, dir, name string, dumpPeriod time.Duration) *DumpWorker {
	ticker := time.NewTicker(dumpPeriod)

	return &DumpWorker{
		setOfNumbers:   setOfNumbers,
		filePath:       filepath.Join(dir, name),
		ticker:         ticker,
		serveStopped:   make(chan struct{}),
		shutdownRunned: make(chan struct{}),
	}
}

func (w *DumpWorker) Serve() error {
	select {
	case _, isOpenned := <-w.serveStopped:
		if !isOpenned {
			return errors.New("buckup worker is stopped")
		}
	default:
		defer close(w.serveStopped)
		defer fmt.Println("stop")
	}

	var stopWait bool
	for !stopWait {
		select {
		case now := <-w.ticker.C:
			log.Println(now)
		case <-w.shutdownRunned:
			stopWait = true
		}
	}

	return nil
}

func (w *DumpWorker) Shutdown() error {
	select {
	case _, isOpenned := <-w.shutdownRunned:
		if !isOpenned {
			return errors.New("buckup worker shutdown already runned")
		}
	default:
		close(w.shutdownRunned)
	}

	w.ticker.Stop()
	<-w.serveStopped

	return nil
}

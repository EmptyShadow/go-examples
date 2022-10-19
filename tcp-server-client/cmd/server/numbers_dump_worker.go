package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type SaveNumbersDump func(ctx context.Context, numbers []int64) error

type NumbersDumpWorker struct {
	setOfNumbers    SetOfNumbers
	saveNumbersDump SaveNumbersDump
	ticker          *time.Ticker
	serveStopped    chan struct{}
	shutdownRunned  chan struct{}
}

func NewNumbersDumpWorker(setOfNumbers SetOfNumbers, saveNumbersDump SaveNumbersDump, dumpPeriod time.Duration) *NumbersDumpWorker {
	return &NumbersDumpWorker{
		setOfNumbers:    setOfNumbers,
		saveNumbersDump: saveNumbersDump,
		ticker:          time.NewTicker(dumpPeriod),
		serveStopped:    make(chan struct{}),
		shutdownRunned:  make(chan struct{}),
	}
}

func (w *NumbersDumpWorker) Serve() error {
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
		case <-w.ticker.C:
			w.dumpNumbers(context.TODO())
		case <-w.shutdownRunned:
			stopWait = true
		}
	}

	return nil
}

func (w *NumbersDumpWorker) dumpNumbers(ctx context.Context) {
	numbers, err := w.readNumbers(ctx)
	if err != nil {
		return
	}

	w.saveNumbersDump(ctx, numbers)
}

func (w *NumbersDumpWorker) readNumbers(ctx context.Context) ([]int64, error) {
	var (
		numbersDump     []int64
		number          int64
		err             error
		streamIsOpenned bool
	)

	w.setOfNumbers.Lock()
	numbers, errs := w.setOfNumbers.StreamOfNumbers(ctx)
	for {
		select {
		case number, streamIsOpenned = <-numbers:
			numbersDump = append(numbersDump, number)
		case err, streamIsOpenned = <-errs:
		}
		if !streamIsOpenned {
			break
		}
	}
	w.setOfNumbers.Unlock()
	if err != nil {
		return nil, fmt.Errorf("read numbers from set: %w", err)
	}

	return numbersDump, nil
}

func (w *NumbersDumpWorker) Shutdown() error {
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

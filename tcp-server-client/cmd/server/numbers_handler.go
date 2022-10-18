package main

import (
	"context"
	"fmt"
	"sync"
)

type SetOfNumbers interface {
	sync.Locker // only for atomic write and read.
	SaveNumber(ctx context.Context, number int64) error
	StreamOfNumbers(ctx context.Context) (<-chan int64, <-chan error)
}

type NumbersHandler struct {
	setOfNumbers SetOfNumbers
}

func NewNumbersHandler(setOfNumbers SetOfNumbers) *NumbersHandler {
	return &NumbersHandler{
		setOfNumbers: setOfNumbers,
	}
}

func (h *NumbersHandler) Handle(ctx context.Context, number int64) (sumOfSquares int64, err error) {
	h.setOfNumbers.Lock()
	defer h.setOfNumbers.Unlock()

	if err := h.setOfNumbers.SaveNumber(ctx, number); err != nil {
		return 0, fmt.Errorf("save number to set: %w", err)
	}

	sumOfSquares, err = h.calculateSumOfSquares(ctx)
	if err != nil {
		return 0, fmt.Errorf("calculate sum of squares: %w", err)
	}

	return sumOfSquares, nil
}

func (h *NumbersHandler) calculateSumOfSquares(ctx context.Context) (sumOfSquares int64, err error) {
	var (
		number         int64
		streamIsClosed bool
	)

	numbers, errs := h.setOfNumbers.StreamOfNumbers(ctx)
	for {
		select {
		case number, streamIsClosed = <-numbers:
			sumOfSquares += number * number
		case err, streamIsClosed = <-errs:
			break
		}
		if streamIsClosed {
			break
		}
	}
	if err != nil {
		return 0, fmt.Errorf("read numbers from set: %w", err)
	}

	return sumOfSquares, nil
}

package main

import (
	"context"
	"sync"
)

var _ SetOfNumbers = (*InmemorySetOfNumbers)(nil)

type InmemorySetOfNumbers struct {
	sync.Mutex
	numbers map[int64]struct{}
}

func NewInmemorySetOfNumbers() *InmemorySetOfNumbers {
	return &InmemorySetOfNumbers{
		numbers: make(map[int64]struct{}),
	}
}

func (s *InmemorySetOfNumbers) SaveNumber(_ context.Context, number int64) error {
	s.numbers[number] = struct{}{}
	return nil
}

func (s *InmemorySetOfNumbers) StreamOfNumbers(ctx context.Context) (<-chan int64, <-chan error) {
	numbersChan := make(chan int64)
	errsChan := make(chan error)

	go func() {
		defer close(numbersChan)
		defer close(errsChan)

		for number := range s.numbers {
			select {
			case <-ctx.Done():
				errsChan <- ctx.Err()
				return
			default:
				numbersChan <- number
			}
		}
	}()

	return numbersChan, errsChan
}

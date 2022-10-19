package main

import (
	"context"
	"sync"
)

type SetOfNumbers interface {
	sync.Locker // only for atomic write and read.
	SaveNumber(ctx context.Context, number int64) error
	StreamOfNumbers(ctx context.Context) (<-chan int64, <-chan error)
}

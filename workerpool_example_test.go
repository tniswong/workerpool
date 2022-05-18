package workerpool_test

import (
	"context"
	"fmt"
	"github.com/tniswong/workerpool"
	"time"
)

func NewCounterTask(name string, limit int) *CounterTask {
	return &CounterTask{name: name, limit: limit}
}

type CounterTask struct {
	name  string
	count int
	limit int
}

func (c *CounterTask) Invoke(ctx context.Context) error {

loop:
	for {

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.Tick(1 * time.Second / 2):

			c.count++
			fmt.Printf("name: %v, count:%v\n", c.name, c.count)

			if c.count >= c.limit {
				break loop
			}

		}

	}

	return nil

}

func Example() {

	wp := workerpool.New(2)
	ctx, cancel := context.WithCancel(context.Background())

	go wp.Run(ctx) // runs until context is cancelled

	wp.Push(NewCounterTask("task 1", 2))
	wp.Push(NewCounterTask("task 2", 3))

	wp.Wait() // blocks until all pending tasks are complete, but does not stop workerpool goroutine
	cancel()  // stops the workerpool

	// Unordered output:
	// name: task 1, count:1
	// name: task 2, count:1
	// name: task 2, count:2
	// name: task 1, count:2
	// name: task 2, count:3

}

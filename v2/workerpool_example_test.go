package workerpool_test

import (
	"context"
	"fmt"
	"github.com/tniswong/workerpool/v2"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool := workerpool.New(2)
	done := pool.Run(ctx) // runs until context is cancelled

	pool.Push(NewCounterTask("task 1", 2))
	pool.Push(NewCounterTask("task 2", 3))

	<-done

	// Unordered output:
	// name: task 1, count:1
	// name: task 2, count:1
	// name: task 2, count:2
	// name: task 1, count:2
	// name: task 2, count:3

}

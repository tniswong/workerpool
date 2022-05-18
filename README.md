# github.com/tniswong/workerpool

![CI](https://github.com/tniswong/workerpool/actions/workflows/ci.yml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/tniswong/workerpool/badge.svg?branch=master)](https://coveralls.io/github/tniswong/workerpool?branch=master)

This package provides a concurrent worker pool implementation using a semaphore for bounded concurrency

```go
package main

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

func main() {

    wp := workerpool.New(2)
    go wp.Run(context.Background())

    wp.Push(NewCounterTask("task 1", 2))
    wp.Push(NewCounterTask("task 2", 3))

    wp.Wait()

    // Unordered output:
    // name: task 1, count:1
    // name: task 2, count:1
    // name: task 2, count:2
    // name: task 1, count:2
    // name: task 2, count:3

}

```
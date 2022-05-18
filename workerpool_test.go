package workerpool_test

import (
	"context"
	"errors"
	"github.com/tniswong/workerpool"
	"testing"
	"time"
)

func assert(t *testing.T, b bool, msg string, args ...any) {
	if !b {
		t.Errorf(msg, args...)
	}
}

type instantlyCompletedTask struct {
	Invocations int
	Completed   bool
}

func (i *instantlyCompletedTask) Invoke(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		i.Invocations++
		i.Completed = true
	}

	return nil

}

type instantErrorTask struct {
	Invocations int
	Completed   bool
}

func (i *instantErrorTask) Invoke(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		i.Invocations++
		i.Completed = true
	}

	return errors.New("why is it always error")
}

type infiniteTask struct {
	Invocations int
}

func (i *infiniteTask) Invoke(ctx context.Context) error {

	if ctx.Err() == nil {
		i.Invocations++
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	}

}

type errorNTimesTask struct {
	N           int
	Invocations int
	Complete    bool
}

func (e *errorNTimesTask) Invoke(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:

		e.Invocations++

		if e.Invocations < e.N {
			return errors.New("why is it always error")
		}

		e.Complete = true

	}

	return nil

}

func TestWorkerPool(t *testing.T) {

	t.Run("With Retry option", func(t *testing.T) {

		ctx := context.Background()

		w := workerpool.New(1)
		go w.Run(ctx)

		expectedInvocations := 5
		task := &errorNTimesTask{N: expectedInvocations}

		w.Push(task, workerpool.Retry(true))

		w.Wait()

		assert(t, expectedInvocations == task.Invocations, "expected '%v' got '%v'", expectedInvocations, task.Invocations)

	})

	t.Run("With RetryMax option", func(t *testing.T) {

		ctx := context.Background()

		w := workerpool.New(1)
		go w.Run(ctx)

		expectedInvocations := 3
		task := &instantErrorTask{}

		w.Push(task, workerpool.RetryMax(expectedInvocations-1))

		w.Wait()

		assert(t, expectedInvocations == task.Invocations, "expected '%v' got '%v'", expectedInvocations, task.Invocations)

	})

	t.Run("With fewer workers than tasks", func(t *testing.T) {

		ctx := context.Background()

		w := workerpool.New(1)
		go w.Run(ctx)

		task1 := &instantlyCompletedTask{}
		task2 := &instantlyCompletedTask{}

		w.Push(task1)
		w.Push(task2)

		w.Wait()

		assert(t, task1.Completed, "task 1 should be complete")
		assert(t, task2.Completed, "task 2 should be complete")

	})

	t.Run("With more workers than tasks", func(t *testing.T) {

		ctx := context.Background()

		w := workerpool.New(3)
		go w.Run(ctx)

		task1 := &instantlyCompletedTask{}
		task2 := &instantlyCompletedTask{}

		w.Push(task1)
		w.Push(task2)

		w.Wait()

		assert(t, task1.Completed, "task 1 should be complete")
		assert(t, task2.Completed, "task 2 should be complete")

	})

	t.Run("With context cancellation", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		w := workerpool.New(2)
		go w.Run(ctx)

		task1 := &infiniteTask{}
		task2 := &infiniteTask{}
		task3 := &infiniteTask{}
		task4 := &infiniteTask{}

		w.Push(task1)
		w.Push(task2)
		w.Push(task3)
		w.Push(task4)

		w.Wait()

		assert(t, task1.Invocations == 1, "task 1 should be invoked [expected %v actual %v]", 1, task1.Invocations)
		assert(t, task2.Invocations == 1, "task 2 should be invoked [expected %v actual %v]", 1, task2.Invocations)
		assert(t, task3.Invocations == 0, "task 3 should not be invoked [expected %v actual %v]", 0, task3.Invocations)
		assert(t, task4.Invocations == 0, "task 4 should not be invoked [expected %v actual %v]", 0, task4.Invocations)

	})

	t.Run("With idling", func(t *testing.T) {

		ctx := context.Background()

		w := workerpool.New(2)
		go w.Run(ctx)

		task1 := &instantlyCompletedTask{}
		task2 := &instantlyCompletedTask{}

		w.Push(task1)
		w.Push(task2)

		w.Wait()

		// idle for 1s
		<-time.Tick(1 * time.Second)

		task3 := &instantlyCompletedTask{}
		task4 := &instantlyCompletedTask{}

		w.Push(task3)
		w.Push(task4)

		w.Wait()

		assert(t, task1.Completed, "task 1 should be complete")
		assert(t, task2.Completed, "task 2 should be complete")
		assert(t, task3.Completed, "task 3 should be complete")
		assert(t, task4.Completed, "task 4 should be complete")

	})

}

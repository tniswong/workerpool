package workerpool

import (
	"context"
	"golang.org/x/sync/semaphore"
	"sync"
)

// TaskOption allows for customization of task behavior
type TaskOption func(j *job)

// Retry will cause a Task to be invoked again if an error is returned. By default, a Task will be retried indefinitely
// until either the Task terminates successfully or the WorkerPool is stopped by context cancellation. To place limits
// on this behavior, see the RetryMax(n int) TaskOption
func Retry(b bool) TaskOption {
	return func(j *job) {
		j.retry = b
	}
}

// RetryMax will cause a Task to be invoked again error is returned, with a limit of `n` total retries
func RetryMax(n int) TaskOption {
	return func(j *job) {
		j.retry = true
		j.retryMax = n
	}
}

// Task represents a unit of work for the WorkerPool
type Task interface {
	Invoke(ctx context.Context) error
}

// job describes a task to be run by the worker pool, and stores details of its execution parameters
type job struct {
	t Task

	retry    bool
	retryMax int
}

// New is a constructor function for WorkerPool. `n` specifies the number of parallel workers
func New(n int64) WorkerPool {

	return WorkerPool{
		mu:  &sync.Mutex{},
		wg:  &sync.WaitGroup{},
		sem: semaphore.NewWeighted(n),

		jobs:   &[]*job{},
		next:   make(chan *job, 1),
		notify: make(chan struct{}),
	}

}

// WorkerPool is a concurrent worker pool implementation based on a semaphore
type WorkerPool struct {
	mu  *sync.Mutex
	wg  *sync.WaitGroup
	sem *semaphore.Weighted

	jobs   *[]*job
	next   chan *job
	notify chan struct{}
}

// work runs a job according to its specified parameters and will notify the WaitGroup and release the semaphore lock
// when completed
func (p *WorkerPool) work(ctx context.Context, j *job) {

	retries := 0

	for {

		err := j.t.Invoke(ctx)

		if err != nil {

			if ctx.Err() != nil {
				break
			}

			if j.retry && (j.retryMax == 0 || retries < j.retryMax) {
				retries++
				continue
			}

		}

		break

	}

	p.wg.Done()
	p.sem.Release(1)

}

// pop is a non-blocking read from the front of the queue
func (p *WorkerPool) pop() *job {

	p.mu.Lock()
	defer p.mu.Unlock()

	// grab next job, if any
	select {
	case j := <-p.next:

		// reload p.next, if possible
		if len(*p.jobs) > 0 {
			select {
			case p.next <- (*p.jobs)[0]:
				*(p.jobs) = (*p.jobs)[1:]
			default:
			}
		}

		return j

	default:
		return nil
	}
}

// Push adds a Task to the queue with the provided options
func (p *WorkerPool) Push(t Task, opts ...TaskOption) {

	p.mu.Lock()
	defer p.mu.Unlock()

	p.wg.Add(1)

	// build the job descriptor for Task t
	j := &job{t: t}
	for _, opt := range opts {
		opt(j)
	}

	// add to queue
	*(p.jobs) = append(*p.jobs, j)

	// reload p.next, if needed
	select {
	case p.next <- (*p.jobs)[0]:
		*(p.jobs) = (*p.jobs)[1:]
	default:
	}

	// notify that new jobs are available
	select {
	case p.notify <- struct{}{}:
	default:
	}

}

// Run the WorkerPool. To stop processing, cancel the context
func (p *WorkerPool) Run(ctx context.Context) {

	go func() {

		for {

			// iterate through jobs in the pool until none are left
			for j := p.pop(); j != nil; j = p.pop() {

				// acquire a semaphore lock. Using context.Background() here rather than ctx because we must still cycle
				// through remaining jobs on the queue once ctx is cancelled.
				_ = p.sem.Acquire(context.Background(), 1)

				// if context is cancelled, mark job as done and release semaphore.
				// this effectively skips the job without running it
				if ctx.Err() != nil {

					p.wg.Done()
					p.sem.Release(1)

					// continuing here allows us to purge the remaining jobs from the queue
					continue

				}

				// run job
				go p.work(ctx, j)

			}

			// blocks until new jobs arrive or the context is cancelled
			select {
			case <-p.notify:
				continue
			case <-ctx.Done():
				return
			}

		}

	}()

}

// Wait blocks until each Task in the queue completes, or the context passed to Run is cancelled
func (p WorkerPool) Wait() {
	p.wg.Wait()
}

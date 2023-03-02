package workerpool

import (
	"context"
	"errors"
	"github.com/semichkin-gopkg/conf"
	"github.com/semichkin-gopkg/promise"
	"sync"
)

var (
	ErrPoolStopped = errors.New("workerpool: pool stopped")
)

type Job[T, R any] struct {
	Payload T
	Promise *promise.Promise[R]
}

type Pool[T, R any] struct {
	configuration Configuration

	workflow func(context.Context, *Job[T, R])
	jobs     chan *Job[T, R]

	poolStoppedCtx    context.Context
	notifyPoolStopped context.CancelFunc

	workersFinishedCtx    context.Context
	notifyWorkersFinished context.CancelFunc

	once sync.Once
}

func NewPool[T, R any](
	workflow func(context.Context, *Job[T, R]),
	updaters ...conf.Updater[Configuration],
) *Pool[T, R] {
	configuration := conf.NewBuilder[Configuration]().
		Append(
			WithWorkersCount(1),
			WithJobsChannelCapacity(256),
		).
		Append(updaters...).
		Build()

	poolStoppedCtx, notifyPoolStopped := context.WithCancel(context.Background())
	workersFinishedCtx, notifyWorkersFinished := context.WithCancel(context.Background())

	return &Pool[T, R]{
		configuration:         configuration,
		workflow:              workflow,
		jobs:                  make(chan *Job[T, R], configuration.JobsChannelCapacity),
		poolStoppedCtx:        poolStoppedCtx,
		notifyPoolStopped:     notifyPoolStopped,
		workersFinishedCtx:    workersFinishedCtx,
		notifyWorkersFinished: notifyWorkersFinished,
	}
}

func (w *Pool[T, R]) Run() {
	var wg sync.WaitGroup
	workersLimitCh := make(chan struct{}, w.configuration.WorkersCount)

F:
	for {
		select {
		case <-w.poolStoppedCtx.Done():
			break F
		case job, ok := <-w.jobs:
			if !ok {
				break F
			}

			workersLimitCh <- struct{}{}

			go func() {
				wg.Add(1)

				defer func() {
					<-workersLimitCh
					wg.Done()
				}()

				w.workflow(w.poolStoppedCtx, job)
			}()
		}
	}

	wg.Wait()
	close(workersLimitCh)
	w.notifyWorkersFinished()
}

func (w *Pool[T, R]) Do(payload T) *promise.Promise[R] {
	task := &Job[T, R]{
		Payload: payload,
		Promise: promise.NewPromise[R](),
	}

	select {
	case <-w.poolStoppedCtx.Done():
		task.Promise.Reject(ErrPoolStopped)
	default:
		w.jobs <- task
	}

	return task.Promise
}

func (w *Pool[T, R]) Stop(ctx context.Context) {
	w.once.Do(func() {
		w.notifyPoolStopped()

		for {
			select {
			case task := <-w.jobs:
				task.Promise.Reject(ErrPoolStopped)
			default:
				close(w.jobs)
				return
			}
		}
	})

	select {
	case <-ctx.Done():
	case <-w.workersFinishedCtx.Done():
	}
}

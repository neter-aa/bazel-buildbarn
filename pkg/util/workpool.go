package util

import (
	"context"
	"sync"
)

type Workpool struct {
	lock                  sync.Mutex
	condEnqueue           *sync.Cond
	condWaitForCompletion *sync.Cond
	context               context.Context
	contextCancelFunc     context.CancelFunc

	firstObservedError error
	maximumConcurrency int
	currentlyRunning   int
	currentlyPending   int
}

func NewWorkpool(ctx context.Context, maximumConcurrency int) *Workpool {
	context, contextCancelFunc := context.WithCancel(ctx)
	w := &Workpool{
		context:            context,
		contextCancelFunc:  contextCancelFunc,
		maximumConcurrency: maximumConcurrency,
	}
	w.condEnqueue = sync.NewCond(&w.lock)
	w.condWaitForCompletion = sync.NewCond(&w.lock)
	return w
}

func (w *Workpool) Enqueue(f func(context.Context) error) {
	// Mark work as pending.
	w.lock.Lock()
	if w.firstObservedError != nil {
		w.lock.Unlock()
		return
	}
	w.currentlyPending++
	w.lock.Unlock()

	go func() {
		// Wait for slot to become available. Bail out if an
		// error occurred in the meantime.
		w.lock.Lock()
		for w.currentlyRunning >= w.maximumConcurrency && w.firstObservedError == nil {
			w.condEnqueue.Wait()
		}
		if w.firstObservedError != nil {
			w.currentlyPending--
			if w.currentlyPending == 0 {
				w.condWaitForCompletion.Broadcast()
			}
			w.lock.Unlock()
			return
		}
		w.currentlyRunning++
		w.lock.Unlock()

		err := f(w.context)

		// Wake up next routine.
		w.lock.Lock()
		w.currentlyRunning--
		if err != nil && w.firstObservedError == nil {
			w.firstObservedError = err
			w.contextCancelFunc()
			w.condEnqueue.Broadcast()
		} else {
			w.condEnqueue.Signal()
		}

		w.currentlyPending--
		if w.currentlyPending == 0 {
			w.condWaitForCompletion.Broadcast()
		}
		w.lock.Unlock()
	}()
}

func (w *Workpool) WaitForCompletion() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	for w.currentlyPending != 0 {
		w.condWaitForCompletion.Wait()
	}
	return w.firstObservedError
}

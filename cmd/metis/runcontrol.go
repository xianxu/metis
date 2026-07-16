package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/xianxu/metis/pkg/experiment"
)

var errRunAborted = errors.New("run aborted after earlier sweep failure")

// runControl bounds admitted concrete runs independently of leaf subprocess
// parallelism and latches the first whole-run failure. Observation callbacks
// must not call back into the controller or block production work.
type runControl struct {
	slots chan struct{}

	mu  sync.Mutex
	err error

	beforeFailureLock   func()
	beforeFailureUnlock func()
	afterAcquire        func(label string)
	beforeRelease       func(label string)
}

func newRunControl(maxParallel int) *runControl {
	control := &runControl{}
	if maxParallel > 1 {
		control.slots = make(chan struct{}, 2*maxParallel)
	}
	return control
}

func (c *runControl) firstError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// whileHealthy linearizes an observable transition against first-failure
// publication. The callback runs while c.mu is held and therefore must not call
// back into runControl. Downstream locks are acquired only inside fn, preserving
// the global control -> progress/pass/manifest order.
func (c *runControl) whileHealthy(fn func()) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return false
	}
	fn()
	return true
}

func (c *runControl) fail(label string, err error) error {
	if err == nil {
		return nil
	}
	contextual := err
	if label != "" {
		contextual = fmt.Errorf("%s: %w", label, err)
	}
	if c.beforeFailureLock != nil {
		c.beforeFailureLock()
	}

	c.mu.Lock()
	if c.err == nil {
		c.err = contextual
		if c.beforeFailureUnlock != nil {
			c.beforeFailureUnlock()
		}
	}
	authoritative := c.err
	c.mu.Unlock()
	return authoritative
}

func (c *runControl) run(label string, fn func() (experiment.Run, error)) (experiment.Run, error) {
	if c.slots != nil {
		c.slots <- struct{}{}
		defer func() { <-c.slots }()
		if c.afterAcquire != nil {
			c.afterAcquire(label)
		}
		if c.beforeRelease != nil {
			defer func() { c.beforeRelease(label) }()
		}
	}

	if c.firstError() != nil {
		return experiment.Run{}, errRunAborted
	}

	run, err := fn()
	if err != nil {
		return experiment.Run{}, c.fail(label, err)
	}
	if c.firstError() != nil {
		return experiment.Run{}, errRunAborted
	}
	return run, nil
}

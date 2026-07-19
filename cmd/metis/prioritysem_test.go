package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPrioritySem_GrantsLowestPriorityFirst: with the budget saturated, freed slots must
// go to the lowest-priority (lowest outer-fold index) waiter first — the property that
// makes fold 0 finish first under --live. We saturate capacity=1 with one holder, queue
// waiters at priorities 3,2,1,0 (in that arrival order, so priority — not FIFO — must
// decide), then release one at a time and record the grant order.
func TestPrioritySem_GrantsLowestPriorityFirst(t *testing.T) {
	s := newPrioritySem(1)
	s.acquire(9) // the initial holder saturates the single slot

	const nWaiters = 4
	prios := []int{3, 2, 1, 0}
	queued := make(chan int, nWaiters)  // reports each waiter once it is definitely enqueued
	granted := make(chan int, nWaiters) // reports the priority of each granted waiter, in grant order
	for _, p := range prios {
		go func(p int) {
			// mark enqueue intent, then block; the test paces releases so all four are
			// queued before the first release (see the wait below).
			queued <- p
			s.acquire(p)
			granted <- p
			s.release()
		}(p)
	}
	// Wait until all four goroutines have reported intent, then give the scheduler a beat
	// so their (blocking) acquire calls have actually pushed onto the heap.
	for i := 0; i < nWaiters; i++ {
		<-queued
	}
	waitForWaiters(t, s, nWaiters)

	s.release() // free the holder's slot → cascade grants best-first as each releases
	var order []int
	for i := 0; i < nWaiters; i++ {
		order = append(order, <-granted)
	}
	want := []int{0, 1, 2, 3}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("grant order = %v, want %v (lowest fold index first)", order, want)
		}
	}
}

// TestPrioritySem_HonorsCapacity: never more than `capacity` holders concurrently, and
// the backfill invariant — a free slot is never idle while a waiter exists (peak
// concurrency reaches capacity). Runs many goroutines and samples live occupancy.
func TestPrioritySem_HonorsCapacity(t *testing.T) {
	const capacity = 4
	s := newPrioritySem(capacity)
	var live int64
	var peak int64
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			s.acquire(p)
			n := atomic.AddInt64(&live, 1)
			for {
				old := atomic.LoadInt64(&peak)
				if n <= old || atomic.CompareAndSwapInt64(&peak, old, n) {
					break
				}
			}
			if n > capacity {
				t.Errorf("live holders %d exceeded capacity %d", n, capacity)
			}
			time.Sleep(200 * time.Microsecond)
			atomic.AddInt64(&live, -1)
			s.release()
		}(i % 7)
	}
	wg.Wait()
	if atomic.LoadInt64(&live) != 0 {
		t.Fatalf("holders leaked: live=%d", atomic.LoadInt64(&live))
	}
	if peak != capacity {
		t.Errorf("peak concurrency = %d, want %d (backfill: a free slot must never idle while leaves wait)", peak, capacity)
	}
	if busy, cap := s.gauge(); busy != 0 || cap != capacity {
		t.Errorf("gauge after drain = (%d,%d), want (0,%d)", busy, cap, capacity)
	}
}

// TestPrioritySem_FIFOWithinPriority: same-priority waiters are granted in arrival order.
func TestPrioritySem_FIFOWithinPriority(t *testing.T) {
	s := newPrioritySem(1)
	s.acquire(0)
	const n = 5
	granted := make(chan int, n)
	for i := 0; i < n; i++ {
		// enqueue strictly in order: acquire the i-th only after i-1 is definitely queued.
		go func(i int) {
			s.acquire(2) // all same priority
			granted <- i
			s.release()
		}(i)
		waitForWaiters(t, s, i+1)
	}
	s.release()
	for i := 0; i < n; i++ {
		if got := <-granted; got != i {
			t.Fatalf("FIFO within priority: grant %d = %d, want %d", i, got, i)
		}
	}
}

// TestChanSem_CountsAndBounds: the default budget bounds concurrency and gauges occupancy.
func TestChanSem_CountsAndBounds(t *testing.T) {
	s := newChanSem(2)
	s.acquire(0)
	s.acquire(5) // priority ignored by the default budget
	if busy, capacity := s.gauge(); busy != 2 || capacity != 2 {
		t.Fatalf("chanSem gauge = (%d,%d), want (2,2)", busy, capacity)
	}
	s.release()
	if busy, _ := s.gauge(); busy != 1 {
		t.Fatalf("chanSem busy after one release = %d, want 1", busy)
	}
	s.release()
}

// waitForWaiters blocks until the sem has exactly `n` queued waiters (the heap length),
// so a test can pace releases against a fully-formed queue without a fixed sleep.
func waitForWaiters(t *testing.T, s *prioritySem, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		s.mu.Lock()
		got := len(s.waiters)
		s.mu.Unlock()
		if got == n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d queued waiters (have %d)", n, got)
		}
		time.Sleep(100 * time.Microsecond)
	}
}

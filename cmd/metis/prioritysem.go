package main

// metis#66: the leaf-subprocess budget as an INTERFACE, so the executor's one global
// budget (metis#31) can either fan leaves out globally (the default) or dispatch them in
// a fold-numbered priority queue (the --live scheduler). The budget is the ONLY budgeted
// resource — a cache HIT never reaches it — so ordering it is the whole of the fold-
// ordered scheduling (no ParExec change: all leaves still fan out as goroutines; the
// budget's grant policy is what makes fold 0 finish first).

import (
	"container/heap"
	"sync"
)

// leafBudget is acquired around every real subprocess spawn and released after it exits.
// priority is the outer-fold index (lower = higher priority); the flat / preamble path
// passes 0. A nil leafBudget means unbounded (the serial / cache-only path) — callers
// nil-check, as they did the pre-#66 `chan struct{}`.
type leafBudget interface {
	acquire(priority int)
	release()
	gauge() (busy, capacity int)
}

// chanSem is the pre-#66 counting semaphore (a `chan struct{}` of cap = maxParallel):
// priority-BLIND, so any waiting leaf may win a freed slot — today's global fan-out, the
// DEFAULT for unattended runs. gauge() reads the live occupancy straight off the channel.
type chanSem struct{ ch chan struct{} }

func newChanSem(capacity int) *chanSem { return &chanSem{ch: make(chan struct{}, capacity)} }

func (s *chanSem) acquire(int) { s.ch <- struct{}{} }
func (s *chanSem) release()    { <-s.ch }
func (s *chanSem) gauge() (int, int) { return len(s.ch), cap(s.ch) }

// prioritySem grants a freed slot to the LOWEST-priority (= lowest outer-fold index)
// waiter first, FIFO within a priority (arrival seq). Two properties fall out:
//   - fold 0 finishes first — its leaves always win the budget → the live estimate
//     tightens fold-by-fold (metis#66 M1);
//   - no idle core (backfill is emergent) — the INVARIANT `len(waiters) > 0 ⟹ inflight
//     == capacity` holds, so a free slot is never held back while a leaf waits; once
//     fold 0's leaves are all in flight, the next-priority ready leaves (fold 1's) take
//     the slack — "backfill" needs no separate logic, just the priority queue.
//
// Correctness of the run is INDEPENDENT of this ordering (the reduce is order-independent,
// metis#18/#31; sortPointRuns normalizes on-disk order) — this only changes WHICH leaf
// runs when, proven byte-identical by the determinism test. Grant order + the backfill
// invariant are locked by the prioritysem unit test (incl. -race).
type prioritySem struct {
	mu       sync.Mutex
	capacity int
	inflight int        // slots currently held (granted, not yet released)
	seq      uint64     // monotone arrival counter → FIFO tiebreak within a priority
	waiters  psWaiterHeap
}

func newPrioritySem(capacity int) *prioritySem { return &prioritySem{capacity: capacity} }

func (s *prioritySem) acquire(priority int) {
	s.mu.Lock()
	// Fast path: a slot is free AND no one is waiting → take it. By the invariant, a
	// non-empty heap implies inflight == capacity, so this can't jump a queued waiter.
	if s.inflight < s.capacity && len(s.waiters) == 0 {
		s.inflight++
		s.mu.Unlock()
		return
	}
	w := &psWaiter{priority: priority, seq: s.seq, ready: make(chan struct{})}
	s.seq++
	heap.Push(&s.waiters, w)
	s.mu.Unlock()
	<-w.ready // Release transferred a slot to us (inflight already accounts for it)
}

func (s *prioritySem) release() {
	s.mu.Lock()
	if len(s.waiters) > 0 {
		// Transfer the slot directly to the best waiter — inflight stays put (the slot
		// never becomes "free" while a waiter exists → the backfill invariant).
		w := heap.Pop(&s.waiters).(*psWaiter)
		s.mu.Unlock()
		close(w.ready)
		return
	}
	s.inflight--
	s.mu.Unlock()
}

func (s *prioritySem) gauge() (busy, capacity int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inflight, s.capacity
}

// psWaiter is one blocked acquirer: its priority (outer-fold index), arrival seq (FIFO
// tiebreak), and the channel Release closes to grant it the transferred slot.
type psWaiter struct {
	priority int
	seq      uint64
	ready    chan struct{}
}

// psWaiterHeap is a min-heap by (priority, seq) — the lowest outer fold, earliest arrival
// first. Only Push/Pop are used (no Fix/Remove), so no back-index bookkeeping is needed.
type psWaiterHeap []*psWaiter

func (h psWaiterHeap) Len() int { return len(h) }
func (h psWaiterHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority < h[j].priority
	}
	return h[i].seq < h[j].seq
}
func (h psWaiterHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *psWaiterHeap) Push(x any)   { *h = append(*h, x.(*psWaiter)) }
func (h *psWaiterHeap) Pop() any {
	old := *h
	n := len(old)
	w := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return w
}

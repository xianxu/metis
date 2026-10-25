package main

import (
	"fmt"
	"io"
)

// Live step log bounds (metis#72): a step's stdout+stderr stream to <stepDir>/step.log as they
// arrive, so an hours-long session is readable while it runs; the in-memory tail feeds the
// failure message (replacing the unbounded CombinedOutput).
const (
	stepLogCap  = 64 << 20 // file bytes kept per step; the rest is drained and dropped
	stepLogTail = 64 << 10 // in-memory tail for the error message
)

const truncationMarker = "\n[metis: step.log truncated at the cap; further output discarded]\n"

// boundedLog is the one writer both launch paths stream child output through. Pure apart from
// its sink: one pass per byte (into the file until the cap, and into a fixed tail ring), memory
// fixed at the tail size, the file at most cap plus one marker. Write always consumes its whole
// input and never returns an error: a failed copier would leave the child blocked on a full pipe.
// A sink error is kept and reported through Err once the step ends. Not safe for concurrent
// Writes; exec.Cmd serializes them when Stdout and Stderr are this same pointer.
type boundedLog struct {
	sink      io.Writer
	capBytes  int64
	written   int64
	truncated bool
	err       error
	ring      []byte
	pos       int  // next ring write index
	full      bool // the ring has wrapped at least once
	consumed  int64
	copied    int64 // byte copies performed, counted for the hot-path test
}

func newBoundedLog(sink io.Writer, capBytes, tail int) *boundedLog {
	return &boundedLog{sink: sink, capBytes: int64(capBytes), ring: make([]byte, tail)}
}

func (l *boundedLog) Write(p []byte) (int, error) {
	l.consumed += int64(len(p))
	if room := l.capBytes - l.written; room > 0 {
		chunk := p
		if int64(len(chunk)) > room {
			chunk = chunk[:room]
		}
		l.put(chunk)
		l.written += int64(len(chunk))
		l.copied += int64(len(chunk))
	}
	if !l.truncated && l.consumed > l.capBytes {
		l.truncated = true
		l.put([]byte(truncationMarker))
	}
	l.remember(p)
	return len(p), nil
}

// put writes to the sink, keeping only the first error.
func (l *boundedLog) put(b []byte) {
	if l.err != nil || len(b) == 0 {
		return
	}
	if _, err := l.sink.Write(b); err != nil {
		l.err = fmt.Errorf("step.log: %w", err)
	}
}

// remember copies at most len(ring) bytes of p into the ring: the tail of p only.
func (l *boundedLog) remember(p []byte) {
	n := len(l.ring)
	if n == 0 {
		return
	}
	if len(p) >= n {
		copy(l.ring, p[len(p)-n:])
		l.pos, l.full = 0, true
		l.copied += int64(n)
		return
	}
	k := copy(l.ring[l.pos:], p)
	if k < len(p) {
		copy(l.ring, p[k:])
		l.full = true
	}
	l.pos = (l.pos + len(p)) % n
	if l.pos == 0 && len(p) > 0 {
		l.full = true
	}
	l.copied += int64(len(p))
}

// Tail returns the last bytes written, oldest first (at most the tail size).
func (l *boundedLog) Tail() []byte {
	if !l.full {
		return append([]byte(nil), l.ring[:l.pos]...)
	}
	return append(append([]byte(nil), l.ring[l.pos:]...), l.ring[:l.pos]...)
}

func (l *boundedLog) Truncated() bool { return l.truncated }
func (l *boundedLog) Err() error      { return l.err }

package main

import (
	"bytes"
	"errors"
	"math/rand"
	"strings"
	"testing"
)

// feed writes total bytes into l in seeded random-sized chunks; returns the input.
func feed(t *testing.T, l *boundedLog, total int, seed int64) []byte {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	in := make([]byte, total)
	for i := range in {
		in[i] = byte('a' + r.Intn(26))
	}
	for off := 0; off < total; {
		n := 1 + r.Intn(97)
		if off+n > total {
			n = total - off
		}
		got, err := l.Write(in[off : off+n])
		if err != nil || got != n {
			t.Fatalf("Write(%d) = %d, %v; a log write must always consume its input", n, got, err)
		}
		off += n
	}
	return in
}

// metis#72 hot-path cost: one pass per byte. At N = cap and 2N = 2·cap the file holds at most
// cap plus one marker, the tail holds the last tailSize bytes, and copies stay ≤ 2·B.
func TestBoundedLog_CountedAtNAnd2N(t *testing.T) {
	const capBytes, tail = 1000, 100
	for _, total := range []int{capBytes, 2 * capBytes} {
		var file bytes.Buffer
		l := newBoundedLog(&file, capBytes, tail)
		in := feed(t, l, total, int64(total))
		if file.Len() > capBytes+len(truncationMarker) {
			t.Errorf("B=%d: file %d bytes, want ≤ cap+marker %d", total, file.Len(), capBytes+len(truncationMarker))
		}
		if got := l.Tail(); !bytes.Equal(got, in[total-tail:]) {
			t.Errorf("B=%d: tail = %q, want the last %d input bytes", total, got, tail)
		}
		if l.copied > int64(2*total) {
			t.Errorf("B=%d: %d byte copies, want ≤ 2·B = %d (a per-write ring re-copy grows worse)", total, l.copied, 2*total)
		}
		if l.consumed != int64(total) {
			t.Errorf("B=%d: consumed %d, want %d", total, l.consumed, total)
		}
		wantTrunc := total > capBytes
		if l.Truncated() != wantTrunc {
			t.Errorf("B=%d: Truncated() = %v, want %v", total, l.Truncated(), wantTrunc)
		}
		if wantTrunc {
			if !bytes.Equal(file.Bytes()[:capBytes], in[:capBytes]) || !strings.HasSuffix(file.String(), truncationMarker) {
				t.Errorf("B=%d: file must be the first cap bytes then one marker", total)
			}
		} else if !bytes.Equal(file.Bytes(), in) {
			t.Errorf("B=%d: under the cap the file must equal the input", total)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) { return 0, errors.New("disk full") }

// A file error must not stop the child: the log keeps consuming (and keeps the tail), and the
// error surfaces once the step ends.
func TestBoundedLog_FileErrorDrainsAndReports(t *testing.T) {
	l := newBoundedLog(failingWriter{}, 1000, 10)
	feed(t, l, 50, 1)
	if l.Err() == nil || !strings.Contains(l.Err().Error(), "disk full") {
		t.Fatalf("Err() = %v, want the file error", l.Err())
	}
	if len(l.Tail()) != 10 {
		t.Errorf("tail must survive a file error, got %d bytes", len(l.Tail()))
	}
}

package sampler

import (
	"testing"

	"github.com/xianxu/metis/pkg/shape"
)

func TestSeqExec_MapsInOrder(t *testing.T) {
	out := SeqExec([]int{1, 2, 3}, func(x int) int { return x * 10 })
	want := []int{10, 20, 30}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out[%d]=%d want %d", i, out[i], want[i])
		}
	}
}

// TestParExec_OrderPreservingUnderReordering forces completion order to be the
// REVERSE of input order: point i blocks until point i+1 has finished. If ParExec
// appended in completion order the result would be reversed; index-addressed writes
// must keep INPUT order.
func TestParExec_OrderPreservingUnderReordering(t *testing.T) {
	pts := []int{0, 1, 2, 3, 4}
	done := make([]chan struct{}, len(pts))
	for i := range done {
		done[i] = make(chan struct{})
	}
	rp := func(x int) int {
		if x+1 < len(pts) {
			<-done[x+1] // wait for my successor to finish first
		}
		res := x * 10
		close(done[x]) // now let my predecessor proceed
		return res
	}
	par := ParExec(pts, rp)
	for i, x := range pts {
		if par[i] != x*10 {
			t.Fatalf("ParExec[%d]=%d want %d (completion-order leak?)", i, par[i], x*10)
		}
	}
}

// TestParExec_RunsConcurrently: N points each block on a shared barrier only all-N
// can release — completes iff they truly run at once (a serial map would hang).
func TestParExec_RunsConcurrently(t *testing.T) {
	const n = 8
	arrived := make(chan struct{}, n)
	release := make(chan struct{})
	out := make(chan []int, 1)
	go func() {
		out <- ParExec(make([]int, n), func(int) int { arrived <- struct{}{}; <-release; return 1 })
	}()
	for i := 0; i < n; i++ {
		<-arrived // all n goroutines reached the barrier ⇒ genuinely concurrent
	}
	close(release)
	if got := <-out; len(got) != n {
		t.Fatalf("got %d results want %d", len(got), n)
	}
}

// TestRun_ParExecEqualsSeqExec (metis#31 M3, sampler-level — closest to the
// Done-when's "byte-identical Done(S)"): the WHOLE reduced SweepResult is identical
// under ParExec vs SeqExec. Order-preserving fan-out + the order-independent reduce
// (metis#18) make this hold with no cmd/metis and no subprocess. Completion order is
// scrambled by the reversal barrier so a completion-order leak would diverge.
func TestRun_ParExecEqualsSeqExec(t *testing.T) {
	names := []string{"logreg", "rf", "gbm", "knn", "svm"}
	pts := make([]shape.Point, len(names))
	for i, n := range names {
		pts[i] = configPoint(n)
	}
	scores := map[string]MeanSE{
		"logreg": meanSE(0.79, "l0", "l1"),
		"rf":     meanSE(0.83, "r0", "r1"), // best
		"gbm":    meanSE(0.81, "g0", "g1"),
		"knn":    meanSE(0.74, "k0", "k1"),
		"svm":    meanSE(0.80, "s0", "s1"),
	}
	idxOf := map[string]int{}
	for i, n := range names {
		idxOf[n] = i
	}
	plain := func(p shape.Point) MeanSE { return scores[p.FreeParams[0].Value.(string)] }
	// Reversal barrier for the PARALLEL run only: config i completes only after config
	// i+1 → completion order is the reverse of input order (a genuine scramble). This
	// would DEADLOCK under a serial exec (point 0 would wait for point 1, which never
	// runs), so the serial baseline uses `plain`; both compute identical scores, so an
	// equal result proves ParExec's order-preservation + the order-independent reduce.
	barrier := func() func(shape.Point) MeanSE {
		done := make([]chan struct{}, len(names))
		for i := range done {
			done[i] = make(chan struct{})
		}
		return func(p shape.Point) MeanSE {
			i := idxOf[p.FreeParams[0].Value.(string)]
			if i+1 < len(names) {
				<-done[i+1]
			}
			ms := scores[p.FreeParams[0].Value.(string)]
			close(done[i])
			return ms
		}
	}
	g := GridConfigs{Points: pts, Direction: "maximize", Select: argmax()}
	seq := Run(Ctx{Seed: 1}, g, plain, SeqExec[shape.Point, MeanSE])
	par := Run(Ctx{Seed: 1}, g, barrier(), ParExec[shape.Point, MeanSE])

	if seq.Ship.Point.FreeParams[0].Value != par.Ship.Point.FreeParams[0].Value {
		t.Fatalf("ship config differs: seq=%v par=%v", seq.Ship.Point.FreeParams, par.Ship.Point.FreeParams)
	}
	if seq.Ship.Score.Mean != par.Ship.Score.Mean || seq.Ship.Score.SE != par.Ship.Score.SE {
		t.Fatalf("winner score differs: seq=%+v par=%+v", seq.Ship.Score, par.Ship.Score)
	}
	if len(seq.PerFamily) != len(par.PerFamily) {
		t.Fatalf("per-family count differs: seq=%d par=%d", len(seq.PerFamily), len(par.PerFamily))
	}
	for fam, sw := range seq.PerFamily {
		pw, ok := par.PerFamily[fam]
		if !ok || sw.Point.FreeParams[0].Value != pw.Point.FreeParams[0].Value || sw.Score.Mean != pw.Score.Mean {
			t.Fatalf("per-family winner for %q differs: seq=%+v par=%+v", fam, sw, pw)
		}
	}
}

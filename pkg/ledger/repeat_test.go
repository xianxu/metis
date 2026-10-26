package ledger

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
)

func intp(i int) *int { return &i }

// A ledger written by an older metis (no repeat, no unknown columns) re-encodes
// byte-identically after Decode, across generated mixes of fold, level, outer-fold,
// stopped and ragged free-param columns: metis#73 changes nothing for legacy data.
func TestEncodeDecode_LegacyRoundTripByteIdentical(t *testing.T) {
	rng := rand.New(rand.NewSource(73))
	for trial := 0; trial < 200; trial++ {
		var l Ledger
		for i := 0; i < 1+rng.Intn(6); i++ {
			r := Row{CodeFingerprint: fmt.Sprintf("fp%d", rng.Intn(2)), PointAddr: fmt.Sprintf("a%d", i), Status: "ok"}
			if rng.Intn(2) == 0 {
				r.FreeParams = map[string]any{"model": []string{"rf", "lr"}[rng.Intn(2)]}
			}
			if rng.Intn(2) == 0 {
				r.Fold = intp(rng.Intn(5))
			}
			if rng.Intn(4) == 0 {
				r.Level, r.OuterFold = "inner", intp(rng.Intn(3))
			}
			if rng.Intn(5) == 0 {
				r.Stopped = "auto"
			}
			if rng.Intn(3) > 0 {
				r.Metrics = map[string]float64{"train.score": rng.Float64()}
			} else {
				r.Status = "failed"
			}
			if err := l.Append(r); err != nil {
				t.Fatal(err)
			}
		}
		b1, err := Encode(l)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Decode(b1)
		if err != nil {
			t.Fatal(err)
		}
		b2, err := Encode(d)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b1, b2) {
			t.Fatalf("trial %d: legacy bytes changed:\n%s\n---\n%s", trial, b1, b2)
		}
		if strings.Contains(string(b1), "repeat") {
			t.Fatalf("a ledger with no repeat rows must not grow a repeat column:\n%s", b1)
		}
	}
}

// A column this version does not know (written by a newer metis) survives
// Decode → Append → Encode with its values on the right rows.
func TestDecode_UnknownColumnsSurviveRewrite(t *testing.T) {
	in := "code_fingerprint,point_addr,status,metric.train.score,zz.future\n" +
		"fp1,a1,ok,0.5,keep-me\n" +
		"fp1,a2,ok,0.6,\n"
	l, err := Decode([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Rows[0].Extra["zz.future"]; got != "keep-me" {
		t.Fatalf("Extra not captured: %v", l.Rows[0].Extra)
	}
	if err := l.Append(Row{CodeFingerprint: "fp1", PointAddr: "a3", Status: "ok", Repeat: "1"}); err != nil {
		t.Fatal(err)
	}
	out, err := Encode(l)
	if err != nil {
		t.Fatal(err)
	}
	want := "code_fingerprint,point_addr,status,repeat,metric.train.score,zz.future\n" +
		"fp1,a1,ok,,0.5,keep-me\n" +
		"fp1,a2,ok,,0.6,\n" +
		"fp1,a3,ok,1,,\n"
	if string(out) != want {
		t.Fatalf("rewrite:\n%s\nwant:\n%s", out, want)
	}
}

func TestDecode_RefusesInvalidRepeatLabel(t *testing.T) {
	_, err := Decode([]byte("code_fingerprint,point_addr,status,repeat\nfp,a,ok,../x\n"))
	if err == nil || !strings.Contains(err.Error(), "row 1") {
		t.Fatalf("want a row-naming refusal, got %v", err)
	}
}

// Two conflicting rows for one repeat execution make the file a contradictory research
// record: Decode reports it, naming the row.
func TestDecode_ConflictingRepeatRowsError(t *testing.T) {
	in := "code_fingerprint,point_addr,status,repeat,metric.s\nfp,a,ok,1,0.5\nfp,a,ok,1,0.7\n"
	if _, err := Decode([]byte(in)); err == nil || !strings.Contains(err.Error(), "conflicting data") {
		t.Fatalf("want a conflict error, got %v", err)
	}
}

// Append and Publish across row kinds × duplicate kinds (metis#73 decision 4 and PQ-2).
func TestAppendPublish_RowKindsByDuplicateKinds(t *testing.T) {
	base := func(kind string) Row {
		r := Row{CodeFingerprint: "fp1", PointAddr: "addr", Status: "ok", Metrics: map[string]float64{"train.s": 0.5}}
		switch kind {
		case "sweep":
			r.Fold = intp(0)
		case "repeat":
			r.Repeat = "1"
		}
		return r
	}
	changed := func(r Row) Row {
		r.Metrics = map[string]float64{"train.s": 0.9}
		return r
	}
	newFP := func(r Row) Row {
		r.CodeFingerprint = "fp2"
		return r
	}
	type want struct {
		rows  int
		err   bool
		score float64 // train.s on the row with fp1
	}
	cases := []struct {
		kind, dup string
		add       func(*Ledger, Row) error
		mk        func(Row) Row
		want      want
	}{
		{"sweep", "identical", (*Ledger).appendOne, func(r Row) Row { return r }, want{1, false, 0.5}},
		{"sweep", "conflicting", (*Ledger).appendOne, changed, want{1, false, 0.5}}, // first wins, unchanged
		{"sweep", "new-fp", (*Ledger).appendOne, newFP, want{2, false, 0.5}},
		{"plain", "identical", (*Ledger).Publish, func(r Row) Row { return r }, want{1, false, 0.5}},
		{"plain", "conflicting", (*Ledger).Publish, changed, want{1, false, 0.9}}, // latest replay wins
		{"plain", "new-fp", (*Ledger).Publish, newFP, want{2, false, 0.5}},
		{"plain", "conflicting-append", (*Ledger).appendOne, changed, want{1, false, 0.5}}, // Append never replaces
		{"repeat", "identical", (*Ledger).Publish, func(r Row) Row { return r }, want{1, false, 0.5}},
		{"repeat", "conflicting", (*Ledger).Publish, changed, want{1, true, 0.5}},
		{"repeat", "conflicting-append", (*Ledger).appendOne, changed, want{1, true, 0.5}},
		{"repeat", "new-fp", (*Ledger).Publish, newFP, want{2, false, 0.5}},
		{"repeat", "other-label", (*Ledger).Publish, func(r Row) Row { r.Repeat = "2"; return r }, want{2, false, 0.5}},
	}
	for _, c := range cases {
		t.Run(c.kind+"/"+c.dup, func(t *testing.T) {
			var l Ledger
			first := base(c.kind)
			if err := c.add(&l, first); err != nil {
				t.Fatal(err)
			}
			err := c.add(&l, c.mk(first))
			if (err != nil) != c.want.err {
				t.Fatalf("err = %v, want err=%v", err, c.want.err)
			}
			if len(l.Rows) != c.want.rows {
				t.Fatalf("rows = %d, want %d", len(l.Rows), c.want.rows)
			}
			if got := l.Rows[0].Metrics["train.s"]; got != c.want.score {
				t.Fatalf("fp1 row score = %v, want %v", got, c.want.score)
			}
		})
	}
}

func (l *Ledger) appendOne(r Row) error { return l.Append(r) }

// An idempotent re-publish survives the CSV round trip: a repeat row read back from disk
// (int free param decoded, NaN metric) equals the freshly derived one.
func TestSameRow_RoundTripStable(t *testing.T) {
	r := Row{CodeFingerprint: "fp", PointAddr: "a", Status: "ok", Repeat: "3",
		FreeParams: map[string]any{"n": 300.0}, Metrics: map[string]float64{"x": math.NaN(), "y": 0.1}}
	var l Ledger
	if err := l.Append(r); err != nil {
		t.Fatal(err)
	}
	b, err := Encode(l)
	if err != nil {
		t.Fatal(err)
	}
	d, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Publish(r); err != nil {
		t.Fatalf("re-publishing the same execution must be a no-op: %v", err)
	}
	if len(d.Rows) != 1 {
		t.Fatalf("rows = %d", len(d.Rows))
	}
}

func repRow(fp, addr, label, status string, m map[string]float64) Row {
	return Row{CodeFingerprint: fp, PointAddr: addr, Repeat: label, Status: status, Metrics: m}
}

func familyOf(t *testing.T, v Ledger, fp, addr string) Row {
	t.Helper()
	for _, r := range v.Rows {
		if r.Repeat == AggregateRepeat && r.CodeFingerprint == fp && r.PointAddr == addr {
			return r
		}
	}
	t.Fatalf("no repeat family for %s/%s in %+v", fp, addr, v.Rows)
	return Row{}
}

// Identical-config repeats reduce to one family row: every metric gets mean, se and n; a
// metric one repeat lacks has a smaller n (never filled); a failed repeat counts in total
// and failed but adds no values.
func TestAggregateView_RepeatFamily(t *testing.T) {
	var l Ledger
	for _, r := range []Row{
		repRow("fp1", "a", "1", "ok", map[string]float64{"train.s": 1, "train.t": 10}),
		repRow("fp1", "a", "2", "ok", map[string]float64{"train.s": 3}),
		repRow("fp1", "a", "3", "failed", map[string]float64{"train.s": 100}),
	} {
		if err := l.Append(r); err != nil {
			t.Fatal(err)
		}
	}
	v := AggregateView(l, "train.s")
	if len(v.Rows) != 1 {
		t.Fatalf("want one family row, got %+v", v.Rows)
	}
	f := familyOf(t, v, "fp1", "a")
	want := map[string]float64{
		"train.s": 2, "train.s.se": 1, "train.s.n": 2,
		"train.t": 10, "train.t.n": 1, // n=1: no se, never an invented zero
		RepeatsTotal: 3, RepeatsFailed: 1,
	}
	for k, w := range want {
		if got, ok := f.Metrics[k]; !ok || math.Abs(got-w) > 1e-12 {
			t.Errorf("%s = %v (present %v), want %v", k, got, ok, w)
		}
	}
	if len(f.Metrics) != len(want) {
		t.Errorf("unexpected metrics: %v", f.Metrics)
	}
	if f.Status != "ok" || f.Fold != nil {
		t.Errorf("family status %q fold %v; want ok, no fold", f.Status, f.Fold)
	}
}

func TestAggregateView_AllFailedFamilyFailed(t *testing.T) {
	var l Ledger
	l.Append(repRow("fp1", "a", "1", "failed", nil), repRow("fp1", "a", "2", "failed", nil))
	f := familyOf(t, AggregateView(l, "train.s"), "fp1", "a")
	if f.Status != "failed" || f.Metrics[RepeatsTotal] != 2 || f.Metrics[RepeatsFailed] != 2 {
		t.Fatalf("family = %+v", f)
	}
	if _, ok := f.Metrics["train.s"]; ok {
		t.Fatal("no value may be invented for an all-failed family")
	}
}

// A code change between repeats gives two families; a plain row of the same point is a
// replay, not a sample, so it passes through beside the family and is not counted.
func TestAggregateView_FingerprintsSplitAndPlainRowPassesThrough(t *testing.T) {
	var l Ledger
	l.Append(
		Row{CodeFingerprint: "fp1", PointAddr: "a", Status: "ok", Metrics: map[string]float64{"train.s": 9}},
		repRow("fp1", "a", "1", "ok", map[string]float64{"train.s": 1}),
		repRow("fp2", "a", "2", "ok", map[string]float64{"train.s": 3}),
	)
	v := AggregateView(l, "train.s")
	if len(v.Rows) != 3 {
		t.Fatalf("want plain + 2 families, got %+v", v.Rows)
	}
	if f := familyOf(t, v, "fp1", "a"); f.Metrics[RepeatsTotal] != 1 || f.Metrics["train.s"] != 1 {
		t.Errorf("fp1 family = %v", f.Metrics)
	}
	if f := familyOf(t, v, "fp2", "a"); f.Metrics[RepeatsTotal] != 1 || f.Metrics["train.s"] != 3 {
		t.Errorf("fp2 family = %v", f.Metrics)
	}
	if v.Rows[0].Repeat != "" || v.Rows[0].Metrics["train.s"] != 9 {
		t.Errorf("plain row changed: %+v", v.Rows[0])
	}
	// The view is idempotent: aggregating it again changes nothing.
	b1, _ := Encode(v)
	b2, _ := Encode(AggregateView(v, "train.s"))
	if !bytes.Equal(b1, b2) {
		t.Fatalf("not idempotent:\n%s\n%s", b1, b2)
	}
}

// Property: for random repeat families, each metric's n equals the number of ok members
// that reported it, and its mean is the mean of exactly those values.
func TestAggregateView_RepeatNCountsReportersOnly(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 200; trial++ {
		var l Ledger
		vals := map[string][]float64{}
		total, failed := 1+rng.Intn(6), 0
		for i := 0; i < total; i++ {
			m := map[string]float64{}
			st := "ok"
			if rng.Intn(4) == 0 {
				st = "failed"
				failed++
			}
			for _, name := range []string{"train.a", "train.b"} {
				if rng.Intn(3) > 0 {
					m[name] = float64(rng.Intn(100))
					if st == "ok" {
						vals[name] = append(vals[name], m[name])
					}
				}
			}
			l.Append(repRow("fp", "a", fmt.Sprintf("r%d", i), st, m))
		}
		f := familyOf(t, AggregateView(l, "train.a"), "fp", "a")
		for _, name := range []string{"train.a", "train.b"} {
			n, has := f.Metrics[name+".n"]
			if len(vals[name]) == 0 {
				if has {
					t.Fatalf("trial %d: %s reported by no ok member but has n=%v", trial, name, n)
				}
				continue
			}
			mean, _ := meanSE(vals[name])
			if int(n) != len(vals[name]) || math.Abs(f.Metrics[name]-mean) > 1e-9 {
				t.Fatalf("trial %d: %s n=%v mean=%v; want %d, %v", trial, name, n, f.Metrics[name], len(vals[name]), mean)
			}
		}
		if int(f.Metrics[RepeatsTotal]) != total || int(f.Metrics[RepeatsFailed]) != failed {
			t.Fatalf("trial %d: counts %v", trial, f.Metrics)
		}
	}
}

// Fold groups keep their pre-metis#73 reduction: adding repeat rows to a fold ledger leaves
// the fold aggregate rows byte-identical (objective-only se, any failure fails the group).
func TestAggregateView_FoldGroupsUnchangedByRepeats(t *testing.T) {
	var folds Ledger
	folds.Append(
		foldRow("fp", "f0", map[string]any{"m": "rf"}, 0, 0.5, "ok"),
		foldRow("fp", "f1", map[string]any{"m": "rf"}, 1, 0.7, "ok"),
		foldRow("fp", "g0", map[string]any{"m": "lr"}, 0, 0.4, "failed"),
	)
	before, _ := Encode(AggregateView(folds, "train.fold_score"))
	mixed := Ledger{Rows: append([]Row(nil), folds.Rows...)}
	mixed.Append(repRow("fp", "r", "1", "ok", map[string]float64{"train.fold_score": 0.9}))
	var foldOnly Ledger
	for _, r := range AggregateView(mixed, "train.fold_score").Rows {
		if r.Repeat == "" {
			foldOnly.Append(r)
		}
	}
	after, _ := Encode(foldOnly)
	if !bytes.Equal(before, after) {
		t.Fatalf("fold aggregates changed:\n%s\n---\n%s", before, after)
	}
}

// Publish refuses a row Decode would refuse (an invalid repeat label), keeps a replaced plain
// row's unknown columns, and an Extra-only difference is no conflict for a repeat re-publish.
func TestPublish_ValidatesLabelAndKeepsExtra(t *testing.T) {
	var l Ledger
	if err := l.Publish(Row{PointAddr: "a", Status: "ok", Repeat: AggregateRepeat}); err == nil {
		t.Fatal("an aggregate marker must not be publishable as a sample")
	}
	plain := Row{CodeFingerprint: "fp", PointAddr: "a", Status: "ok", Extra: map[string]string{"zz.new": "v"}}
	rep := Row{CodeFingerprint: "fp", PointAddr: "a", Status: "ok", Repeat: "1", Extra: map[string]string{"zz.new": "w"}}
	if err := l.Append(plain, rep); err != nil {
		t.Fatal(err)
	}
	if err := l.Publish(Row{CodeFingerprint: "fp", PointAddr: "a", Status: "failed"}); err != nil {
		t.Fatal(err)
	}
	if l.Rows[0].Status != "failed" || l.Rows[0].Extra["zz.new"] != "v" {
		t.Fatalf("replace must keep unknown columns: %+v", l.Rows[0])
	}
	if err := l.Publish(Row{CodeFingerprint: "fp", PointAddr: "a", Status: "ok", Repeat: "1"}); err != nil {
		t.Fatalf("a re-derived repeat row (no Extra) must be idempotent: %v", err)
	}
	if !l.Has(rep) || l.Has(Row{CodeFingerprint: "fp", PointAddr: "a", Repeat: "2"}) {
		t.Fatal("Has must follow the row key")
	}
}

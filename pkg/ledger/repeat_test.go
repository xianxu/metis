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

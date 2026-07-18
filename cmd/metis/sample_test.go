package main

import "testing"

func TestParseSample(t *testing.T) {
	cases := []struct {
		in         string
		out, inner int
		wantErr    bool
	}{
		{"out1", 1, 0, false},
		{"out3", 3, 0, false},
		{"in2", 0, 2, false},
		{"out1in2", 1, 2, false},
		{"out10in5", 10, 5, false},
		{"", 0, 0, false},       // unset — no sampling
		{"3", 0, 0, true},       // bare integer retired (breaking, by design)
		{"out0", 0, 0, true},    // zero is not a fold count
		{"in0", 0, 0, true},
		{"out", 0, 0, true},     // missing number
		{"in", 0, 0, true},
		{"outin2", 0, 0, true},
		{"in2out1", 0, 0, true}, // fixed order: out before in
		{"out1in2x", 0, 0, true},
		{"OUT1", 0, 0, true},    // lowercase only
		{"out99999999999999999999", 0, 0, true}, // overflow must be loud, not silently-unset
	}
	for _, c := range cases {
		got, err := parseSample(c.in)
		if c.wantErr != (err != nil) {
			t.Errorf("parseSample(%q): err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if err == nil && (got.Out != c.out || got.In != c.inner) {
			t.Errorf("parseSample(%q) = %+v, want {Out:%d In:%d}", c.in, got, c.out, c.inner)
		}
	}
}

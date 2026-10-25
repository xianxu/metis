package main

import (
	"strings"
	"testing"
	"time"
)

// metis#72: every row of the outcome table, plus the precedence
// cleanup-unconfirmed > resources-unretired > timed-out/cancelled/failed > ok.
func TestClassifyOutcome_Table(t *testing.T) {
	t0 := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	ran := func(ev endEvent, code int, sig string, joined bool, pending string) stepEnding {
		return stepEnding{event: ev, launched: true, exitCode: code, signal: sig, joined: joined,
			pending: []byte(pending), started: t0, ended: t0.Add(90 * time.Second), tail: []byte("last words")}
	}
	cases := []struct {
		name    string
		in      stepEnding
		want    string
		errHas  string // "" = nil error
	}{
		{"exit 0", ran(endExited, 0, "", true, ""), outcomeOK, ""},
		{"exit nonzero", ran(endExited, 3, "", true, ""), outcomeFailed, "exit status 3"},
		{"deadline TERM", ran(endDeadline, -1, "TERM", true, ""), outcomeTimedOut, "timed out"},
		{"deadline KILL", ran(endDeadline, -1, "KILL", true, ""), outcomeTimedOut, "KILL"},
		{"cancelled running", ran(endCancelled, -1, "TERM", true, ""), outcomeCancelled, "cancelled"},
		{"cancelled before launch", stepEnding{event: endCancelled}, outcomeCancelled, "before launch"},
		{"join unconfirmed beats exit 0", ran(endExited, 0, "", false, ""), outcomeCleanupUnconfirmed, "cleanup unconfirmed"},
		{"join unconfirmed beats pending", ran(endDeadline, -1, "KILL", false, `["actor-1"]`), outcomeCleanupUnconfirmed, "cleanup unconfirmed"},
		{"pending beats exit 0", ran(endExited, 0, "", true, `["actor-1"]`), outcomeResourcesUnretired, `["actor-1"]`},
		{"pending beats timeout", ran(endDeadline, -1, "TERM", true, `["actor-1"]`), outcomeResourcesUnretired, "pending-resources.json"},
		{"pending beats failure", ran(endExited, 2, "", true, `["actor-1"]`), outcomeResourcesUnretired, "actor-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := classifyOutcome(c.in)
			if got.Outcome != c.want {
				t.Errorf("outcome = %q, want %q", got.Outcome, c.want)
			}
			if c.errHas == "" {
				if err != nil {
					t.Errorf("err = %v, want nil (only ok publishes success)", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.errHas) {
				t.Errorf("err = %v, want one containing %q", err, c.errHas)
			}
		})
	}
}

// Only a launched step carries times; runtime is launch → exit, and failure messages carry the tail.
func TestClassifyOutcome_RecordFields(t *testing.T) {
	t0 := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	o, err := classifyOutcome(stepEnding{event: endExited, launched: true, exitCode: 1, joined: true,
		started: t0, ended: t0.Add(1500 * time.Millisecond), tail: []byte("boom"), logTruncated: true})
	if o.RuntimeSeconds == nil || *o.RuntimeSeconds != 1.5 || o.Started == nil || o.ExitCode == nil || *o.ExitCode != 1 || !o.LogTruncated {
		t.Errorf("record = %+v", o)
	}
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("failure error must carry the log tail, got %v", err)
	}
	pre, _ := classifyOutcome(stepEnding{event: endCancelled})
	if pre.Started != nil || pre.RuntimeSeconds != nil || pre.ExitCode != nil {
		t.Errorf("a never-launched step has no times or exit code: %+v", pre)
	}
}

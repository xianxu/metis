// sample.go — the --sample grammar (metis#58): out<M>, in<N>, out<M>in<N>.
// M subsamples the OUTER folds, N the INNER per-config folds; both are
// deterministic prefix subsets of the shape-declared partitions (the shape's
// k/inner_k stay the estimand — the flag only trades precision for cost).
// The bare-integer form (--sample 3) is retired: one grammar, parsed here only.
package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type sampleSpec struct {
	Out int // outer folds to run; 0 = all k
	In  int // inner folds per config; 0 = all inner_k
}

var sampleRe = regexp.MustCompile(`^(?:out([1-9][0-9]*))?(?:in([1-9][0-9]*))?$`)

func parseSample(s string) (sampleSpec, error) {
	if s == "" {
		return sampleSpec{}, nil
	}
	m := sampleRe.FindStringSubmatch(s)
	if m == nil || (m[1] == "" && m[2] == "") {
		return sampleSpec{}, fmt.Errorf(
			"--sample %q: want out<M>, in<N>, or out<M>in<N> (e.g. --sample out1in2; M,N ≥ 1; the bare-integer form is retired — use out<M>)", s)
	}
	// strconv.Atoi, NOT Sscanf: an overflowing count (out99999999999999999999) must be a
	// loud error, not a silently-unset spec that runs the FULL sweep.
	var sp sampleSpec
	var err error
	if m[1] != "" {
		if sp.Out, err = strconv.Atoi(m[1]); err != nil {
			return sampleSpec{}, fmt.Errorf("--sample %q: out count: %v", s, err)
		}
	}
	if m[2] != "" {
		if sp.In, err = strconv.Atoi(m[2]); err != nil {
			return sampleSpec{}, fmt.Errorf("--sample %q: in count: %v", s, err)
		}
	}
	return sp, nil
}

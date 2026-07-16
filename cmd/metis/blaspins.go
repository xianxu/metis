package main

import "strings"

// blasPinDefaults are the single-thread pins metis injects into LEAF subprocesses by
// default (metis#48): the parallelism budget belongs to the ORCHESTRATOR (the metis#31
// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
// oversubscribes ~NumCPU× (observed: load-avg 83 on 12 cores, throughput ≈ 0).
//
// Cache identity: env is deliberately OUTSIDE run identity — Kpre hashes
// {step_id, uses, with, seed, upstream} (pkg/cache), HIT-validation re-hashes the
// read-set D (file blob hashes), and the code fingerprint is git state. Injecting
// pins perturbs neither cache keys nor fingerprints — exactly as the RUNBOOK's
// manual `OMP_NUM_THREADS=1 metis run` never did.
var blasPinDefaults = []string{
	"MKL_NUM_THREADS=1",
	"OMP_NUM_THREADS=1",
	"OPENBLAS_NUM_THREADS=1",
	"VECLIB_MAXIMUM_THREADS=1",
}

// blasPins returns the defaults NOT already set in environ — an explicit operator
// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
// passes through untouched). Pure. Always non-nil: an all-suppressed result is
// empty, distinguishable from runOpts' nil "not yet computed" sentinel.
func blasPins(environ []string) []string {
	pins := make([]string, 0, len(blasPinDefaults))
	for _, def := range blasPinDefaults {
		name := def[:strings.IndexByte(def, '=')]
		if !envHasName(environ, name) {
			pins = append(pins, def)
		}
	}
	return pins
}

// envHasName reports whether environ sets exactly `name` (match up to '=').
func envHasName(environ []string, name string) bool {
	for _, kv := range environ {
		if strings.HasPrefix(kv, name) && len(kv) > len(name) && kv[len(name)] == '=' {
			return true
		}
	}
	return false
}

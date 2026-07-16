package main

import (
	"reflect"
	"testing"
)

// TestBlasPins_BareEnv: no ambient thread vars → all four pins injected, sorted.
func TestBlasPins_BareEnv(t *testing.T) {
	got := blasPins([]string{"PATH=/usr/bin", "HOME=/h"})
	want := []string{
		"MKL_NUM_THREADS=1",
		"OMP_NUM_THREADS=1",
		"OPENBLAS_NUM_THREADS=1",
		"VECLIB_MAXIMUM_THREADS=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("blasPins = %v, want %v", got, want)
	}
}

// TestBlasPins_OperatorValueWins: an ambient-set var is NOT pinned — an explicit
// operator choice always wins (the issue's escape hatch by construction).
func TestBlasPins_OperatorValueWins(t *testing.T) {
	got := blasPins([]string{"OMP_NUM_THREADS=8", "PATH=/usr/bin"})
	for _, kv := range got {
		if kv == "OMP_NUM_THREADS=1" {
			t.Fatalf("ambient OMP_NUM_THREADS=8 must suppress the pin; got %v", got)
		}
	}
	if len(got) != 3 {
		t.Errorf("want 3 remaining pins, got %v", got)
	}
}

// TestBlasPins_AllSetIsEmptyNonNil: fully pinned ambient env → empty but NON-nil
// (runExperiment uses nil as "not yet computed"; empty must not recompute).
func TestBlasPins_AllSetIsEmptyNonNil(t *testing.T) {
	got := blasPins([]string{
		"OMP_NUM_THREADS=4", "OPENBLAS_NUM_THREADS=4",
		"VECLIB_MAXIMUM_THREADS=4", "MKL_NUM_THREADS=4",
	})
	if got == nil || len(got) != 0 {
		t.Errorf("want empty non-nil, got %#v (nil=%v)", got, got == nil)
	}
}

// TestBlasPins_PrefixNotName: OMP_NUM_THREADS_X=9 is a DIFFERENT var — must not
// suppress the OMP_NUM_THREADS pin (name match is exact, up to '=').
func TestBlasPins_PrefixNotName(t *testing.T) {
	got := blasPins([]string{"OMP_NUM_THREADS_X=9"})
	found := false
	for _, kv := range got {
		if kv == "OMP_NUM_THREADS=1" {
			found = true
		}
	}
	if !found {
		t.Errorf("prefix-named var must not suppress the real pin; got %v", got)
	}
}

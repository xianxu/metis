package cas

import (
	"bytes"
	"testing"
)

// MemStore's defensive copy on Put and Get is load-bearing: consumers must not be
// able to mutate stored bytes through an alias to their input or a returned slice.
func TestMemStore_CopiesInsulateStoredBytes(t *testing.T) {
	s := NewMemStore()
	in := []byte("original")
	h, err := s.Put(in)
	if err != nil {
		t.Fatal(err)
	}

	// Mutating the caller's input slice after Put must not change what's stored.
	in[0] = 'X'
	got, err := s.Get(h)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("original")) {
		t.Errorf("stored bytes changed via input alias: got %q", got)
	}

	// Mutating a returned slice must not change what's stored either.
	got[0] = 'Y'
	again, err := s.Get(h)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, []byte("original")) {
		t.Errorf("stored bytes changed via returned alias: got %q", again)
	}
}

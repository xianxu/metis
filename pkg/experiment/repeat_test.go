package experiment

import "testing"

func TestParseRepeatLabel(t *testing.T) {
	for _, ok := range []string{"1", "r-2", "baseline", "a234567890123456789012345678901b"} {
		if got, err := ParseRepeatLabel(ok); err != nil || got != ok {
			t.Errorf("ParseRepeatLabel(%q) = %q, %v; want ok", ok, got, err)
		}
	}
	for _, bad := range []string{"", "*", "../x", "a/b", "A", "x y", "a2345678901234567890123456789012c"} {
		if _, err := ParseRepeatLabel(bad); err == nil {
			t.Errorf("ParseRepeatLabel(%q) accepted; want refusal", bad)
		}
	}
}

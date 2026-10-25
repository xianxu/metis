package experiment

import (
	"fmt"
	"regexp"
)

// repeatLabelRE is the repeat-label grammar (metis#73): a short slug, so a label is safe as
// a run-dir name component and can never equal the ledger's reserved aggregate marker "*".
var repeatLabelRE = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)

// ParseRepeatLabel validates a repeat label: the execution coordinate `metis run --repeat`
// gives one execution of an experiment, so repeated executions stay distinct samples.
func ParseRepeatLabel(s string) (string, error) {
	if !repeatLabelRE.MatchString(s) {
		return "", fmt.Errorf("invalid repeat label %q: want 1 to 32 of [a-z0-9-]", s)
	}
	return s, nil
}

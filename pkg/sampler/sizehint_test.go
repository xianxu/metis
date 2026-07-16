package sampler

import (
	"testing"

	"github.com/xianxu/metis/pkg/shape"
)

// SizeHint is the per-sampler "n" (metis#30): every production sampler is a static
// point-set, so all four return their exact cardinality.
func TestSizeHints(t *testing.T) {
	grid := GridConfigs{Points: make([]shape.Point, 7)}
	if n, k := grid.SizeHint(grid.Init(Ctx{})); n != 7 || k != SizeExact {
		t.Errorf("grid: (%d,%v)", n, k)
	}
	folds := FixedKFolds{K: 5}
	if n, k := folds.SizeHint(folds.Init(Ctx{})); n != 5 || k != SizeExact {
		t.Errorf("folds: (%d,%v)", n, k)
	}
	cv := CVDriver{K: 3}
	if n, k := cv.SizeHint(cv.Init(Ctx{})); n != 3 || k != SizeExact {
		t.Errorf("cv: (%d,%v)", n, k)
	}
	var sd SingleDriver
	if n, k := sd.SizeHint(sd.Init(Ctx{})); n != 1 || k != SizeExact {
		t.Errorf("single: (%d,%v)", n, k)
	}
}

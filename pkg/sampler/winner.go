package sampler

import "github.com/xianxu/metis/pkg/shape"

// Winner is the sweeper's Done: the selected config as reconstructable run-keys — the
// FULL resolved config Point (its per-step `With` + free-params), the seed, and the fold
// addresses that scored it (the winning config's resample told-set). Ship (M1a-5) and
// nested-CV (metis#23) rebuild the exact run DIRECTLY from Point.With — not by re-expanding
// the grid and matching free-params. Point.FreeParams are the human-legible run-keys (single
// source — no separate FreeParams field to drift). Score carries the winning (mean, SE).
type Winner struct {
	Point    shape.Point
	Seed     int
	FoldKeys []string
	Score    MeanSE
	Family   string // the model-family key this winner belongs to (metis#19; "" = one implicit family)
}

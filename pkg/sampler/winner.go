package sampler

import "github.com/xianxu/metis/pkg/shape"

// Winner is the sweeper's Done: the selected config as reconstructable run-keys —
// its free-params + seed + the fold addresses that scored it (the winning config's
// resample told-set) — so ship/assessment (M1a-5) and nested-CV (metis#23) rebuild
// the exact run faithfully, not just abstract hyperparameters. Score carries the
// winning (mean, SE).
type Winner struct {
	FreeParams []shape.FreeParam
	Seed       int
	FoldKeys   []string
	Score      MeanSE
}

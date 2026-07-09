package sampler

import (
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// stat builds a hand-made ConfigStat identified by its `id` free-param, in a family,
// with a per-config (mean, complexity). ToldSet gives std a defined n for mean-std.
func stat(family, id string, mean, cx float64) ConfigStat {
	return ConfigStat{
		Point:  shape.Point{FreeParams: []shape.FreeParam{{Path: "id", Value: id}}},
		Family: family,
		Score:  MeanSE{Mean: mean, MeanComplexity: cx, HasComplexity: true, ToldSet: []string{"f0", "f1", "f2", "f3", "f4"}},
	}
}

func idOf(w Winner) string {
	if len(w.Point.FreeParams) == 0 {
		return ""
	}
	s, _ := w.Point.FreeParams[0].Value.(string)
	return s
}

func rule(s experiment.Select) experiment.Select { return s }

// argmax-mean: the family/ship winner is the highest mean; complexity ignored.
func TestSelect_ArgmaxMean(t *testing.T) {
	stats := []ConfigStat{
		stat("rf", "depth8", 0.844, 40),
		stat("rf", "depth4", 0.834, 16),
	}
	res := SelectConfigs(rule(experiment.Select{ArgmaxMean: &experiment.ArgmaxMean{}}), "maximize", 42, stats)
	if idOf(res.Ship) != "depth8" {
		t.Errorf("argmax-mean ship = %q, want depth8 (highest mean)", idOf(res.Ship))
	}
	if idOf(res.PerFamily["rf"]) != "depth8" {
		t.Errorf("rf per-family winner = %q, want depth8", idOf(res.PerFamily["rf"]))
	}
	if res.Ship.Seed != 42 {
		t.Errorf("ship seed = %d, want 42", res.Ship.Seed)
	}
}

// THE CORNER REGRESSION: argmax-mean picks the deep overfitter (depth8), but pct-loss
// keeps the shallower configs (lower complexity) in the band and — since both depth4s
// tie on complexity — the mean tie-break lands on the more-feature depth4/feat6, NOT
// the sparse depth4/feat1 corner and NOT the deep depth8.
func TestSelect_PctLoss_TieBreaksToMean(t *testing.T) {
	stats := []ConfigStat{
		stat("rf", "depth8_feat3", 0.844, 40), // argmax-mean winner (overfit)
		stat("rf", "depth4_feat6", 0.834, 16), // the empirically-good config
		stat("rf", "depth4_feat1", 0.830, 16), // sparse; same complexity, lower CV
	}
	res := SelectConfigs(rule(experiment.Select{PctLoss: &experiment.PctLoss{Tolerance: 0.02}}), "maximize", 1, stats)
	if got := idOf(res.Ship); got != "depth4_feat6" {
		t.Errorf("pct-loss ship = %q, want depth4_feat6 (band admits depth4s; cx-tie → mean picks feat6)", got)
	}
}

// ε-binning: complexities within complexityBinRelTol (0.10) are "equally simple", so
// the mean tie-break can still operate; outside it, min-complexity wins outright.
func TestSelect_PctLoss_BinnedComplexity(t *testing.T) {
	// feat1 cx 15, feat6 cx 16 → 16 ≤ 15·1.10 = 16.5 → tie → mean picks feat6.
	within := []ConfigStat{
		stat("rf", "feat6", 0.834, 16),
		stat("rf", "feat1", 0.830, 15),
	}
	res := SelectConfigs(rule(experiment.Select{PctLoss: &experiment.PctLoss{Tolerance: 0.05}}), "maximize", 1, within)
	if got := idOf(res.Ship); got != "feat6" {
		t.Errorf("within-ε: ship = %q, want feat6 (cx 15~16 tie → mean)", got)
	}
	// feat1 cx 10, feat6 cx 16 → 16 > 10·1.10 = 11 → feat1 strictly simpler → feat1 wins.
	outside := []ConfigStat{
		stat("rf", "feat6", 0.834, 16),
		stat("rf", "feat1", 0.830, 10),
	}
	res = SelectConfigs(rule(experiment.Select{PctLoss: &experiment.PctLoss{Tolerance: 0.05}}), "maximize", 1, outside)
	if got := idOf(res.Ship); got != "feat1" {
		t.Errorf("outside-ε: ship = %q, want feat1 (cx 10 << 16 → strictly simpler)", got)
	}
}

// one-std-err's SE-width band is too tight here: with SE ~0.005 the 1-SE floor
// excludes a config 0.006 below the best, so it can't reach it (documents the
// 15×-too-tight finding — pct-loss decouples from SE).
func TestSelect_OneStdErr_BandTooTight(t *testing.T) {
	best := stat("rf", "deep", 0.844, 40)
	best.Score.SE = 0.005
	near := stat("rf", "shallow", 0.838, 16) // 0.006 below best > 1×SE
	near.Score.SE = 0.005
	res := SelectConfigs(rule(experiment.Select{OneStdErr: &experiment.OneStdErr{}}), "maximize", 1, []ConfigStat{best, near})
	if got := idOf(res.Ship); got != "deep" {
		t.Errorf("one-std-err ship = %q, want deep (shallow is below the tight 1-SE floor)", got)
	}
}

// mean-std re-scores by mean − λ·std and ignores complexity: a high-variance config
// loses to a slightly-lower-mean but stabler one.
func TestSelect_MeanStd_UsesStd_NotComplexity(t *testing.T) {
	fragile := stat("rf", "fragile", 0.850, 8) // high mean, high variance, LOW complexity
	fragile.Score.SE = 0.02                    // std = 0.02·√5 ≈ 0.0447
	stable := stat("rf", "stable", 0.840, 40)  // lower mean, zero variance, HIGH complexity
	stable.Score.SE = 0.0
	// mean−1·std: fragile 0.850−0.0447=0.805; stable 0.840−0=0.840 → stable wins.
	res := SelectConfigs(rule(experiment.Select{MeanStd: &experiment.MeanStd{Lambda: 1.0}}), "maximize", 1, []ConfigStat{fragile, stable})
	if got := idOf(res.Ship); got != "stable" {
		t.Errorf("mean-std ship = %q, want stable (penalized variance beats it; complexity irrelevant)", got)
	}
}

// minimize pct-loss (e.g. a loss/error objective): the band admits configs within
// tolerance ABOVE the best (lowest) mean, then parsimony picks the simpler one — the
// minimize analogue of the corner regression. best-mean `low_deep` (cx 40) is in-band
// but complex; `mid_shallow` (mean 0.102 = best·1.02, cx 16) wins on parsimony.
func TestSelect_PctLoss_MinimizeDirection(t *testing.T) {
	stats := []ConfigStat{
		stat("rf", "low_deep", 0.100, 40),    // best (lowest) mean, deep/complex
		stat("rf", "mid_shallow", 0.102, 16), // within 2% band (0.100·1.02=0.102), simpler
		stat("rf", "high", 0.150, 8),         // out of band despite lowest complexity
	}
	res := SelectConfigs(rule(experiment.Select{PctLoss: &experiment.PctLoss{Tolerance: 0.02}}), "minimize", 1, stats)
	if got := idOf(res.Ship); got != "mid_shallow" {
		t.Errorf("minimize pct-loss ship = %q, want mid_shallow (band admits it; simpler than low_deep; high is out of band)", got)
	}
}

// minimize mean-std: for minimize the variance penalty is ADDED (mean + λ·std), so a
// fragile config with a lower raw mean but high variance loses to a stabler one.
func TestSelect_MeanStd_MinimizeDirection(t *testing.T) {
	fragile := stat("rf", "fragile", 0.100, 8) // lowest raw mean, high variance
	fragile.Score.SE = 0.02                    // std = 0.02·√5 ≈ 0.0447 → adj 0.1447
	stable := stat("rf", "stable", 0.110, 40)  // higher mean, zero variance → adj 0.110
	stable.Score.SE = 0.0
	res := SelectConfigs(rule(experiment.Select{MeanStd: &experiment.MeanStd{Lambda: 1.0}}), "minimize", 1, []ConfigStat{fragile, stable})
	if got := idOf(res.Ship); got != "stable" {
		t.Errorf("minimize mean-std ship = %q, want stable (fragile's variance penalty pushes its adjusted loss above stable)", got)
	}
}

// cross-family: each family selects its own robust winner; the ship pick is
// argmax-mean over those winners (never a cross-family complexity comparison).
func TestSelect_CrossFamily_ArgmaxMean(t *testing.T) {
	stats := []ConfigStat{
		stat("rf", "rf_a", 0.834, 16),
		stat("rf", "rf_b", 0.844, 40),
		stat("logreg", "lr_a", 0.820, 6),
		stat("logreg", "lr_b", 0.810, 3),
	}
	res := SelectConfigs(rule(experiment.Select{PctLoss: &experiment.PctLoss{Tolerance: 0.02}}), "maximize", 1, stats)
	if len(res.PerFamily) != 2 {
		t.Fatalf("per-family count = %d, want 2", len(res.PerFamily))
	}
	// rf winner mean ≥ logreg winner mean → ship is from rf.
	if res.Ship.Family != "rf" {
		t.Errorf("ship family = %q, want rf (higher-mean family winner)", res.Ship.Family)
	}
}

// a shape with no tagged sum is one implicit family (empty family key).
func TestSelect_NoTaggedSum_OneImplicitFamily(t *testing.T) {
	stats := []ConfigStat{stat("", "a", 0.80, 5), stat("", "b", 0.85, 9)}
	res := SelectConfigs(rule(experiment.Select{ArgmaxMean: &experiment.ArgmaxMean{}}), "maximize", 1, stats)
	if len(res.PerFamily) != 1 {
		t.Errorf("implicit-family count = %d, want 1", len(res.PerFamily))
	}
	if idOf(res.Ship) != "b" {
		t.Errorf("ship = %q, want b", idOf(res.Ship))
	}
}

// FamilyOf reads the tagged-sum branch label off Point.With's single-key-map bundling
// — and must NOT treat a swept UNTAGGED bare-string alternative (whose With value is a
// bare string, not a map) as a family. This is the exact tagged-vs-untagged
// disambiguation the spec flagged as hard.
func TestFamilyOf(t *testing.T) {
	// tagged sum: With["train"]["model"] == {"rf": {...}}, FreeParam Value "rf".
	tagged := shape.Point{
		With:       map[string]map[string]any{"train": {"model": map[string]any{"rf": map[string]any{"max_depth": 8}}}},
		FreeParams: []shape.FreeParam{{Path: "train.model", Value: "rf"}},
	}
	if got := FamilyOf(tagged); got != "train.model=rf" {
		t.Errorf("FamilyOf(tagged) = %q, want train.model=rf", got)
	}
	// untagged bare-string alternative ($any: ["a","b"]): With value is a bare string.
	bareString := shape.Point{
		With:       map[string]map[string]any{"features": {"features": "a"}},
		FreeParams: []shape.FreeParam{{Path: "features.features", Value: "a"}},
	}
	if got := FamilyOf(bareString); got != "" {
		t.Errorf("FamilyOf(bare-string) = %q, want \"\" (an untagged string alt is NOT a family)", got)
	}
	// untagged list alternative ($any: [[], [title]]): Value is a list, not a string.
	list := shape.Point{
		With:       map[string]map[string]any{"features": {"features": []any{"title"}}},
		FreeParams: []shape.FreeParam{{Path: "features.features", Value: []any{"title"}}},
	}
	if got := FamilyOf(list); got != "" {
		t.Errorf("FamilyOf(list) = %q, want \"\"", got)
	}
	// no free params → implicit single family.
	if got := FamilyOf(shape.Point{}); got != "" {
		t.Errorf("FamilyOf(empty) = %q, want \"\"", got)
	}
}

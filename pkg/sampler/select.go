package sampler

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// GuardComplexity rejects a parsimony rule (one-std-err / pct-loss) when ANY swept family's
// configs lack a measured complexity — a silently-dropped parsimony axis would give a
// quietly-wrong winner (the metis#19 v1 corner failure, one level up). Checked post-fold,
// pre-selection: HasComplexity is only knowable after folds run (the metric is emitted by
// the model step, not statically declared), so there is no pre-sweep registry to consult.
// Checks EVERY family (not just the eventual winner — each family's within-winner needs
// complexity). argmax-mean / mean-std never read complexity → never trip it.
func GuardComplexity(rule experiment.Select, stats []ConfigStat) error {
	if rule.OneStdErr == nil && rule.PctLoss == nil {
		return nil // only parsimony rules consume complexity
	}
	var missing []string
	seen := map[string]bool{}
	for _, s := range stats {
		if !s.Score.HasComplexity && !seen[s.Family] {
			seen[s.Family] = true
			missing = append(missing, s.Family)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	kind, _ := rule.Kind()
	return fmt.Errorf("select rule %q needs a measured complexity, but families %v report none — "+
		"have each model class emit a `complexity` metric (metis.model.complexity(kind)) or use argmax-mean/mean-std",
		kind, missing)
}

// complexityBinRelTol: two complexities within this relative tolerance are "equally
// simple" (the same band idea pct-loss applies to the score) — so the mean tie-break
// can recover a higher-CV config whose realized complexity is ~equal. 0.10 absorbs a
// small integer leaf-count difference (e.g. 15 vs 16 leaves = 6.7% ties). Plan-pinned;
// tuned in the metis#19 acceptance. If retuned, update TestSelect_PctLoss_BinnedComplexity.
const complexityBinRelTol = 0.10

// ConfigStat is one config's reduced statistics the select rule reasons over: its
// resolved Point, its model family (the tagged-sum branch label), and the per-config
// (mean, SE, mean-complexity) reduction. Built by GridConfigs.Done (in-memory) and by
// the offline ledger path (metis#19 M2) — the shared input that lets one pure rule
// serve both selection surfaces (ARCH-DRY).
type ConfigStat struct {
	Point  shape.Point
	Family string
	Score  MeanSE
}

// SweepResult is the select rule's output (metis#19): each family's robust winner plus
// the single cross-family ship pick (argmax-mean over the per-family winners).
type SweepResult struct {
	PerFamily map[string]Winner
	Ship      Winner
}

// SelectConfigs applies the select rule. Group by family; within each family the rule's
// band sets the contention set, then parsimony (minimize measured complexity, ε-binned)
// → tie-break by mean; the cross-family Ship = argmax-mean over the per-family winners
// (never a cross-family complexity comparison — incommensurable units, metis#19 §A).
// Pure: a re-reduction over the cached per-config stats, no IO.
func SelectConfigs(rule experiment.Select, direction string, seed int, stats []ConfigStat) SweepResult {
	byFam := map[string][]ConfigStat{}
	var order []string // first-seen family order → deterministic
	for _, s := range stats {
		if _, seen := byFam[s.Family]; !seen {
			order = append(order, s.Family)
		}
		byFam[s.Family] = append(byFam[s.Family], s)
	}
	perFamily := make(map[string]Winner, len(order))
	var winners []ConfigStat
	for _, fam := range order {
		w, ok := selectWithinFamily(rule, direction, byFam[fam])
		if !ok {
			continue
		}
		perFamily[fam] = toWinner(w, seed)
		winners = append(winners, w)
	}
	ship, _ := argmaxMeanStat(direction, winners)
	return SweepResult{PerFamily: perFamily, Ship: toWinner(ship, seed)}
}

// selectWithinFamily runs the rule over one family's configs → that family's winner.
func selectWithinFamily(rule experiment.Select, direction string, fam []ConfigStat) (ConfigStat, bool) {
	switch {
	case rule.MeanStd != nil:
		return argmaxMeanStd(direction, rule.MeanStd.Lambda, fam)
	case rule.OneStdErr != nil:
		best, ok := argmaxMeanStat(direction, fam)
		if !ok {
			return ConfigStat{}, false
		}
		return parsimony(direction, withinBand(direction, best.Score.Mean, best.Score.SE, fam))
	case rule.PctLoss != nil:
		best, ok := argmaxMeanStat(direction, fam)
		if !ok {
			return ConfigStat{}, false
		}
		return parsimony(direction, withinBand(direction, best.Score.Mean, math.Abs(best.Score.Mean)*rule.PctLoss.Tolerance, fam))
	default: // argmax-mean (also the M1a behavior)
		return argmaxMeanStat(direction, fam)
	}
}

// argmaxMeanStat picks the best-mean config; strict comparison keeps the FIRST among
// equals (stable tie-break = earliest in slice order).
func argmaxMeanStat(direction string, cands []ConfigStat) (ConfigStat, bool) {
	best := -1
	for i := range cands {
		if best < 0 || betterMean(cands[i].Score.Mean, cands[best].Score.Mean, direction) {
			best = i
		}
	}
	if best < 0 {
		return ConfigStat{}, false
	}
	return cands[best], true
}

// argmaxMeanStd re-scores each config by mean − λ·std (penalizing fold-to-fold
// fragility) and picks the best; for "minimize" the penalty is added. Ignores complexity.
func argmaxMeanStd(direction string, lambda float64, cands []ConfigStat) (ConfigStat, bool) {
	adj := func(c ConfigStat) float64 {
		std := c.Score.SE * math.Sqrt(float64(len(c.Score.ToldSet)))
		if direction == "minimize" {
			return c.Score.Mean + lambda*std
		}
		return c.Score.Mean - lambda*std
	}
	best := -1
	for i := range cands {
		if best < 0 || betterMean(adj(cands[i]), adj(cands[best]), direction) {
			best = i
		}
	}
	if best < 0 {
		return ConfigStat{}, false
	}
	return cands[best], true
}

// withinBand returns the configs whose mean is within `width` of the family best (the
// contention set): mean ≥ best−width for maximize, ≤ best+width for minimize.
func withinBand(direction string, bestMean, width float64, fam []ConfigStat) []ConfigStat {
	var out []ConfigStat
	for _, c := range fam {
		in := c.Score.Mean >= bestMean-width
		if direction == "minimize" {
			in = c.Score.Mean <= bestMean+width
		}
		if in {
			out = append(out, c)
		}
	}
	return out
}

// parsimony picks the simplest config in the band — minimize measured complexity, but
// with an ε-relative bin so near-equal complexities tie — then breaks ties by mean.
func parsimony(direction string, band []ConfigStat) (ConfigStat, bool) {
	if len(band) == 0 {
		return ConfigStat{}, false
	}
	minCx := math.Inf(1)
	for _, c := range band {
		if c.Score.MeanComplexity < minCx {
			minCx = c.Score.MeanComplexity
		}
	}
	// ε-bin around the minimum; guard non-positive minima (all-zero → all tie, the M1
	// wired-as-0 case) so the threshold never shrinks below the minimum.
	threshold := minCx
	if minCx > 0 {
		threshold = minCx * (1 + complexityBinRelTol)
	}
	var simplest []ConfigStat
	for _, c := range band {
		if c.Score.MeanComplexity <= threshold {
			simplest = append(simplest, c)
		}
	}
	return argmaxMeanStat(direction, simplest)
}

// toWinner wraps a winning ConfigStat with the sweep seed → the reconstructable Winner.
func toWinner(s ConfigStat, seed int) Winner {
	return Winner{
		Point:    s.Point,
		Seed:     seed,
		FoldKeys: s.Score.ToldSet,
		Score:    s.Score,
		Family:   s.Family,
	}
}

// familySelect picks the model FAMILY from per-family honest OUTER estimates (metis#32): among
// families whose mean is within 1 SE of the best family's mean, the LOWEST-SE (most stable) one.
// It deliberately does NOT reuse SweepResult.Ship (the cross-family inner-argmax that ships the
// overfitter #32 exists to replace) — the family choice rides on the honest OUTER estimate.
//
// Degrades to argmax-mean when SE is unavailable (a single outer fold — `metis run --fast`). The
// caveat states the honesty gloss: a 1-SE pick over N families is ~mildly optimistic — and it's
// the *family's* estimate, not a per-config claim. ok=false iff est is empty.
func FamilySelect(direction string, est map[string]MeanSE) (family, caveat string, ok bool) {
	if len(est) == 0 {
		return "", "", false
	}
	var bestFam string
	var best MeanSE
	for fam, ms := range est {
		if bestFam == "" || betterMean(ms.Mean, best.Mean, direction) {
			bestFam, best = fam, ms
		}
	}
	anySE := false
	for _, ms := range est {
		if ms.SE > 0 {
			anySE = true
			break
		}
	}
	if !anySE {
		return bestFam, fmt.Sprintf("single outer fold — no SE, so argmax-mean over %d families (honest but high-variance); run the full nested `metis run` (drop --fast) for the SE-based rule", len(est)), true
	}
	// Band = families within 1 SE of the best family's mean; pick the lowest-SE (most stable —
	// also penalizes inner-selection instability for free). Reuses the withinBand idiom over families.
	width := best.SE
	pick, pickSE := bestFam, best.SE
	for fam, ms := range est {
		inBand := ms.Mean >= best.Mean-width
		if direction == "minimize" {
			inBand = ms.Mean <= best.Mean+width
		}
		if inBand && ms.SE < pickSE {
			pick, pickSE = fam, ms.SE
		}
	}
	return pick, fmt.Sprintf("the selected family's honest estimate is a 1-SE pick over %d families (~mildly optimistic — the family's estimate, not the shipped config's)", len(est)), true
}

// FamilyOf reads a Point's model family — the set of tagged-sum ($any-map) branch
// labels — off its resolved `With` bundling. A tagged sum resolves to a single-key map
// `{label: sub}` at its step.key path (shape.Expand), so a FreeParam whose Value is the
// sole key of that map is a family discriminant. An untagged bare-string alternative
// (`$any: ["a","b"]`) has a bare string in `With`, not a map → NOT a family; a list/range
// alternative has a non-string Value → skipped. Empty = one implicit family.
//
// EXPORTED so BOTH selection surfaces derive the SAME family key: the in-memory
// GridConfigs.Done and the offline `metis ledger select` path (which matches an aggregate
// row to its Expanded Point, then calls this) — otherwise the two would key the same config
// differently and the one-rule-two-surfaces DRY property breaks (metis#19 M1-review).
//
// M1 reads a 2-level `With[step][key]` — sufficient for the swept `train.model`; a
// deeper-nested tagged sum is out of scope.
func FamilyOf(p shape.Point) string {
	var parts []string
	for _, fp := range p.FreeParams {
		label, ok := fp.Value.(string)
		if !ok {
			continue // range (float) / untagged list (slice) alternative — not a tag
		}
		seg := strings.SplitN(fp.Path, ".", 2)
		if len(seg) != 2 {
			continue
		}
		stepWith, ok := p.With[seg[0]]
		if !ok {
			continue
		}
		m, ok := stepWith[seg[1]].(map[string]any)
		if !ok || len(m) != 1 {
			continue // bare string (untagged alt) or not single-key → not a tagged discriminant
		}
		if _, isBranch := m[label]; !isBranch {
			continue
		}
		parts = append(parts, seg[0]+"."+seg[1]+"="+label)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

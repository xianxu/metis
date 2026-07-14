package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
)

// cmdLedger handles `metis ledger show <shape.md> [--fingerprint HASH] [--sort metric] [--top N]`
// — renders the shape's append-only ledger sidecar as a sorted/filtered VIEW (the CSV
// stays append-order; sorting is never a storage concern).
func cmdLedger(args []string) error {
	if len(args) == 0 || args[0] != "show" {
		return fmt.Errorf("usage: metis ledger show <shape.md> [flags] (selection moved to `metis select`, metis#32)")
	}
	fs := flag.NewFlagSet("ledger show", flag.ContinueOnError)
	fingerprint := fs.String("fingerprint", "", "filter to one code-fingerprint (code-version, metis#27)")
	sortMetric := fs.String("sort", "", "sort by this namespaced metric (e.g. train.fold_score)")
	direction := fs.String("dir", "", "sort direction: maximize | minimize (default: the shape's objective direction)")
	top := fs.Int("top", 0, "show only the top N (0 = all)")
	shapePath, flags, err := hoistShapePath(args[1:])
	if err != nil {
		return fmt.Errorf("ledger show: %w (usage: metis ledger show <shape.md> [--fingerprint HASH] [--sort metric] [--top N])", err)
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	// Default the sort direction from the shape's objective (so `--sort` on a minimize
	// objective sorts best-first, not descending). Explicit --dir overrides.
	dir := *direction
	if dir == "" {
		dir = "maximize"
		if raw, err := os.ReadFile(shapePath); err == nil {
			if sh, err := experiment.ParseShape(string(raw)); err == nil && sh.Sweeper.Objective.Direction != "" {
				dir = sh.Sweeper.Objective.Direction
			}
		}
	}
	return showLedger(shapePath, *fingerprint, *sortMetric, dir, *top, os.Stdout)
}

// showLedger is the testable core of `ledger show`: load, filter, sort/top, render — to
// any io.Writer (so the rendered table can be asserted against a buffer).
func showLedger(shapePath, fingerprint, sortMetric, direction string, top int, out io.Writer) error {
	led, err := loadLedger(shapePath)
	if err != nil {
		return err
	}
	led = ledger.Filter(led, fingerprint)
	rows := led.Rows
	if sortMetric != "" {
		// metis#18: the sidecar holds RAW per-fold rows — reduce to per-config (mean, SE)
		// before ranking, so `--sort <metric>` is a config leaderboard, not fold noise.
		agg := ledger.AggregateView(led, sortMetric)
		rows = ledger.SortAll(agg, sortMetric, direction) // sorts by objective, KEEPS failed/missing rows
	}
	if top > 0 && top < len(rows) {
		rows = rows[:top]
	}
	renderLedger(out, rows)
	return nil
}

// hoistShapePath pulls the single `<shape.md>` positional out of args and returns it
// plus the remaining flag args — so flags may appear before OR after the path (Go's
// stdlib flag stops at the first positional, which broke the documented
// `metis <cmd> <shape.md> --flags` order). The positional is the lone `.md`-suffixed
// arg; every other non-flag token is a flag value (e.g. `--point train.model=rf`).
func hoistShapePath(args []string) (shapePath string, flags []string, err error) {
	for _, a := range args {
		if strings.HasSuffix(a, ".md") && !strings.HasPrefix(a, "-") {
			if shapePath != "" {
				return "", nil, fmt.Errorf("want exactly one <shape.md>, got multiple")
			}
			shapePath = a
			continue
		}
		flags = append(flags, a)
	}
	if shapePath == "" {
		return "", nil, fmt.Errorf("missing <shape.md>")
	}
	return shapePath, flags, nil
}

// renderLedger prints rows as a table (a header row + code-fingerprint short, status,
// free-params, metrics) to any io.Writer.
func renderLedger(out io.Writer, rows []ledger.Row) {
	if len(rows) == 0 {
		fmt.Fprintln(out, "(no rows)")
		return
	}
	metricCols := map[string]bool{}
	for _, r := range rows {
		for k := range r.Metrics {
			metricCols[k] = true
		}
	}
	mCols := make([]string, 0, len(metricCols))
	for k := range metricCols {
		mCols = append(mCols, k)
	}
	sort.Strings(mCols)
	fmt.Fprintln(out, strings.Join(append([]string{"code", "status", "free_params"}, mCols...), "  "))
	for _, r := range rows {
		parts := []string{short(r.CodeFingerprint), r.Status, freeParamTuple(r)}
		for _, c := range mCols {
			if v, ok := r.Metrics[c]; ok {
				parts = append(parts, fmt.Sprintf("%s=%g", c, v))
			} else {
				parts = append(parts, "")
			}
		}
		fmt.Fprintln(out, strings.Join(parts, "  "))
	}
}

// short renders a git SHA as its 8-char eyeballable prefix.
func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// freeParamTupleMap renders a {path: value} free-param map as `(k=v, …)`, keys sorted.
// Kept for `ledger show`'s freeParamTuple (metis#32 retired promote, which also used it).
func freeParamTupleMap(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%v", k, m[k])
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

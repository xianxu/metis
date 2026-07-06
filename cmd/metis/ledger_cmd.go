package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
)

// cmdLedger handles `metis ledger show <shape.md> [--sweep SHA] [--sort metric] [--top N]`
// — renders the shape's append-only ledger sidecar as a sorted/filtered VIEW (the CSV
// stays append-order; sorting is never a storage concern).
func cmdLedger(args []string) error {
	if len(args) == 0 || args[0] != "show" {
		return fmt.Errorf("usage: metis ledger show <shape.md> [--sweep SHA] [--sort metric] [--top N]")
	}
	fs := flag.NewFlagSet("ledger show", flag.ContinueOnError)
	sweep := fs.String("sweep", "", "filter to one sweep-SHA (code-version)")
	sortMetric := fs.String("sort", "", "sort by this namespaced metric (e.g. train.cv_score)")
	direction := fs.String("dir", "", "sort direction: maximize | minimize (default: the shape's objective direction)")
	top := fs.Int("top", 0, "show only the top N (0 = all)")
	shapePath, flags, err := hoistShapePath(args[1:])
	if err != nil {
		return fmt.Errorf("ledger show: %w (usage: metis ledger show <shape.md> [--sweep SHA] [--sort metric] [--top N])", err)
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
			if sh, err := experiment.ParseShape(string(raw)); err == nil && sh.Sweep.Objective.Direction != "" {
				dir = sh.Sweep.Objective.Direction
			}
		}
	}
	return showLedger(shapePath, *sweep, *sortMetric, dir, *top, os.Stdout)
}

// showLedger is the testable core of `ledger show`: load, filter, sort/top, render — to
// any io.Writer (so the rendered table can be asserted against a buffer).
func showLedger(shapePath, sweep, sortMetric, direction string, top int, out io.Writer) error {
	led, err := loadLedger(shapePath)
	if err != nil {
		return err
	}
	led = ledger.Filter(led, sweep)
	rows := led.Rows
	if sortMetric != "" {
		rows = ledger.SortAll(led, sortMetric, direction) // sorts by objective, KEEPS failed/missing rows
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

// renderLedger prints rows as a table (a header row + sweep-SHA short, status,
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
	fmt.Fprintln(out, strings.Join(append([]string{"sweep", "status", "free_params"}, mCols...), "  "))
	for _, r := range rows {
		parts := []string{short(r.SweepSHA), r.Status, freeParamTuple(r)}
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

// cmdPromote handles `metis promote <shape.md> (--best | --point 'k=v,...') [--sweep SHA]
// --name X` — selects a ledger row, reconstructs its all-singleton experiment, writes
// <name>.md with a back-link, and commits it at the code SHA (warns if dirty).
func cmdPromote(args []string) error {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	best := fs.Bool("best", false, "promote the whole-ledger champion by the shape's objective")
	point := fs.String("point", "", "promote the row matching these free-params (e.g. 'train.model=rf')")
	sweep := fs.String("sweep", "", "restrict selection to one sweep-SHA")
	name := fs.String("name", "", "output experiment name (writes <name>.md)")
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("promote: %w (usage: metis promote <shape.md> (--best | --point 'k=v,..') [--sweep SHA] --name X)", err)
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if *name == "" || (!*best && *point == "") {
		return fmt.Errorf("usage: metis promote <shape.md> (--best | --point 'k=v,..') [--sweep SHA] --name X")
	}
	return runPromote(promoteOpts{
		shapePath: shapePath, best: *best, point: *point, sweep: *sweep, name: *name,
		out: os.Stdout, git: gitCLI{}, commit: gitCLICommitter{},
	})
}

// gitCLICommitter is the production gitCommitter: it shells `git -C <dir> add/commit`.
type gitCLICommitter struct{}

func (gitCLICommitter) Add(dir, path string) error {
	_, err := gitOut(dir, "add", "--", path)
	return err
}

func (gitCLICommitter) Commit(dir, msg string) error {
	_, err := gitOut(dir, "commit", "-m", msg)
	return err
}

type promoteOpts struct {
	shapePath string
	best      bool
	point     string
	sweep     string
	name      string
	out       io.Writer
	git       gitProbe
	commit    gitCommitter // nil → skip the commit (tests without a repo); cmdPromote injects the real one
}

// gitCommitter commits a file at the current SHA (injected so promote is testable
// without a real repo when needed; the real impl shells git).
type gitCommitter interface {
	Add(dir, path string) error
	Commit(dir, msg string) error
}

func runPromote(o promoteOpts) error {
	raw, err := os.ReadFile(o.shapePath)
	if err != nil {
		return err
	}
	sh, err := experiment.ParseShape(string(raw))
	if err != nil {
		return err
	}
	led, err := loadLedger(o.shapePath)
	if err != nil {
		return err
	}
	led = ledger.Filter(led, o.sweep)

	var row ledger.Row
	if o.best {
		best, ok := ledger.Best(led, sh.Sweep.Objective.Metric, sh.Sweep.Objective.Direction)
		if !ok {
			return fmt.Errorf("promote --best: no qualifying row for objective %q — metrics are namespaced `<step>.<metric>` (e.g. train.%s); check the shape's sweep.objective.metric", sh.Sweep.Objective.Metric, sh.Sweep.Objective.Metric)
		}
		row = best
	} else {
		r, ok := findRow(led, parsePointSelector(o.point))
		if !ok {
			return fmt.Errorf("promote --point %q: no ledger row matches those free-params", o.point)
		}
		row = r
	}

	// A promoted winner from an OLDER code-version (its sweep-SHA ≠ HEAD) would be
	// committed at HEAD, so its code isn't the code it was measured under — the promoted
	// experiment wouldn't reproduce the row without `git checkout <its sweep-SHA>` first
	// (the design's deliberate "go back"). Warn loudly rather than silently mis-attribute.
	if _, headSHA, _ := probeRepo(o.git, filepath.Dir(o.shapePath)); row.SweepSHA != "" && headSHA != "" && row.SweepSHA != headSHA {
		fmt.Fprintf(o.out, "metis: warning: the selected row ran at code %s but HEAD is %s — the promoted %s is committed at HEAD, so it reproduces only after `git checkout %s`\n", short(row.SweepSHA), short(headSHA), o.name, short(row.SweepSHA))
	}

	exp, err := promotedExperiment(sh, row.FreeParams)
	if err != nil {
		return err
	}
	exp.ID = o.name // the promoted experiment's id matches its <name>.md filename (the experiment convention)
	doc := renderPromoted(exp, sh.ID, row)
	outPath := filepath.Join(filepath.Dir(o.shapePath), o.name+".md")
	if err := os.WriteFile(outPath, []byte(doc), 0o644); err != nil {
		return err
	}

	// Commit the promoted experiment at the code SHA — a self-contained reproducible
	// commit. Warn if the repo is dirty (a promoted winner should be commit-nameable;
	// metis#8 M3's side-ref capture makes even a dirty run's SHA real).
	dir := filepath.Dir(o.shapePath)
	if o.commit != nil {
		if _, _, dirty := probeRepo(o.git, dir); dirty {
			fmt.Fprintf(o.out, "metis: warning: repo is dirty — committing %s against a dirty tree\n", o.name)
		}
		if err := o.commit.Add(dir, filepath.Base(outPath)); err != nil {
			return err
		}
		if err := o.commit.Commit(dir, "metis promote: "+o.name); err != nil {
			return err
		}
		fmt.Fprintf(o.out, "metis: promoted + committed %s → %s\n", freeParamTupleMap(row.FreeParams), outPath)
	} else {
		fmt.Fprintf(o.out, "metis: promoted %s → %s (not committed)\n", freeParamTupleMap(row.FreeParams), outPath)
	}
	return nil
}

// findRow returns the ledger row whose free-params equal the selector (JSON-tolerant,
// so int/float CSV drift doesn't break the match).
func findRow(led ledger.Ledger, want map[string]any) (ledger.Row, bool) {
	wb, _ := json.Marshal(want)
	for _, r := range led.Rows {
		if rb, _ := json.Marshal(r.FreeParams); bytes.Equal(rb, wb) {
			return r, true
		}
	}
	return ledger.Row{}, false
}

// short renders a git SHA as its 8-char eyeballable prefix.
func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// parsePointSelector parses `k=v,k2=v2` into a free-param map (values typed via the
// same round-trip as the CSV, so a selector matches a decoded row).
func parsePointSelector(s string) map[string]any {
	m := map[string]any{}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if i := strings.Index(kv, "="); i > 0 {
			m[strings.TrimSpace(kv[:i])] = ledgerParseCell(strings.TrimSpace(kv[i+1:]))
		}
	}
	return m
}

// renderPromoted writes the all-singleton experiment markdown with a back-link that
// records the FULL origin provenance — the shape, the row's point-address, its
// sweep-SHA (the code-version it was measured under), and the free-param tuple — so the
// promoted experiment can be checked against (and recovered to) its origin row.
func renderPromoted(exp experiment.Experiment, fromShape string, row ledger.Row) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "type: experiment\nid: %s\n", exp.ID)
	if exp.Competition != "" {
		fmt.Fprintf(&b, "competition: %s\n", exp.Competition)
	}
	fmt.Fprintf(&b, "seed: %d\nstatus: active\n", exp.Seed)
	fmt.Fprintf(&b, "promoted_from: %s @ %s (sweep %s) %s\n", fromShape, row.PointAddr, short(row.SweepSHA), freeParamTupleMap(row.FreeParams))
	b.WriteString("steps:\n")
	for _, s := range exp.Steps {
		fmt.Fprintf(&b, "  - id: %s\n    uses: %s\n", s.ID, s.Uses)
		if len(s.Needs) > 0 {
			fmt.Fprintf(&b, "    needs: [%s]\n", strings.Join(s.Needs, ", "))
		}
		wb, _ := yamlInline(s.With)
		fmt.Fprintf(&b, "    with: %s\n", wb)
	}
	b.WriteString("---\n\n# " + exp.ID + "\n\nPromoted from the `" + fromShape + "` sweep.\n")
	return b.String()
}

// freeParamTupleMap renders a {path: value} free-param map as `(k=v, …)`, keys sorted.
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

// ledgerParseCell types a selector value the same way the CSV codec does, so a
// `--point 'k=v'` selector matches a decoded row's free-param value.
func ledgerParseCell(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// yamlInline renders a `with` map as an inline YAML mapping (JSON is valid YAML flow
// syntax, so the promoted experiment re-parses cleanly).
func yamlInline(m map[string]any) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

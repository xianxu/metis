package record

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/metis/internal/repo"
)

// repoRoot returns the metis module root (nearest ancestor go.mod), so the drift
// guard addresses construct/vocabulary/ the same way regardless of where `go test`
// runs. Thin wrapper over the shared repo.Root (ARCH-DRY).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := repo.Root(wd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestRunRecordConformsToCUE is the ARCH-DRY drift guard for the provenance record:
// the Go RunRecord/StepRecord structs restate the CUE #RunRecord (construct/
// vocabulary/experiment.cue, the single structural source). Like #Run there's no
// markdown fixture (a record is emitted as record.json with no `type` discriminator),
// so this marshals a representative RunRecord to JSON and `cue vet`s it against the
// closed #RunRecord — a renamed/removed/extra field would fail, so the struct can't
// silently drift. SKIPS when `cue` is unavailable, mirroring the #Run guard.
func TestRunRecordConformsToCUE(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH; skipping #RunRecord drift guard")
	}
	root := repoRoot(t)
	cueFile := filepath.Join(root, "construct", "vocabulary", "experiment.cue")

	// A representative record exercising every field (incl. the #2-populated slots
	// d/deps and the optional upstream/output_hash/metrics) so the closed schema is
	// fully checked.
	rec := RunRecord{
		RunID:        "run-001",
		Experiment:   "titanic-baseline",
		Seed:         42,
		PointAddress: "abc123",
		RepoSHAs:     map[string]string{"metis": "deadbeef", "kbench": "cafef00d"},
		Dirty:        false,
		Steps: []StepRecord{{
			StepID:   "cv-split",
			Uses:     "metis/cv-split",
			With:     map[string]any{"k": 5, "shuffle": true},
			Upstream: []Hash{"u1", "u2"},
			Code: CodeManifest{
				Commit: "deadbeef",
				Dirty:  false,
				D:      []CodeRef{{Path: "steps/metis/cv-split", BlobHash: "b1"}},
				Deps:   "uvlock-digest",
			},
			OutputHash: "oh1",
			Metrics:    map[string]float64{"cv_score": 0.81},
		}},
		Started:  "2026-07-01T00:00:00Z",
		Finished: "2026-07-01T00:00:05Z",
		Status:   "ok",
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "record.json")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("cue", "vet", "-d", "#RunRecord", tmp, cueFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cue vet rejected a valid RunRecord against #RunRecord (CUE drift?): %v\n%s", err, out)
	}
}

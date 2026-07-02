# metis atlas — index

metis is the **platform-independent ML workbench** — the base layer of the
`kaggle-ml-base-layer` stack (`kbench → kaggle → metis → ariadne`). It owns the
reproducible unit of ML work (the **experiment**) and, as they land, the step-runner
and the Dataset/Split/step-type data plane. "Platform-independent" test: *would this be
identical on a non-Kaggle platform?* — if yes, it lives here.

- [experiment datatype](experiment.md) — the reproducible pipeline noun: the CUE schema
  (`#Experiment`/`#Step`/`#Status`/`#Run`), the `xx-datatype` authoring prototype, the
  `vocabulary validate-instance` structural validator, the enforcement merge-check (M1), the
  Go step-runner (M2), and the Python data plane — Dataset/Schema/Split + `cv-split`/`train`/
  `predict` step-types run hermetically via uv (M3).
- [workflow/](workflow) — inherited ariadne workflow docs (symlink into the substrate).

**Roadmap (metis#1 — all milestones shipped):** M1 (experiment datatype), M2 (Go step-runner:
`cmd/metis run` + pure `pkg/experiment` `Parse`/`Validate`/`TopoSort`, semantics enforced on
read, steps run as subprocesses over a `steps/<layer>/<steptype>` + files contract), and M3
(Python data plane: pure `metis/` core + thin `metis/io` contract + `cv-split`/`train`/`predict`
entrypoints + uv env; `metis run` walks a toy pipeline to a real CV score). The end-to-end
Kaggle proof is kbench's Titanic walking skeleton (kbench#1), which builds on this base.

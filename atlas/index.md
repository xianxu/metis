# metis atlas — index

metis is the **platform-independent ML workbench** — the base layer of the
`kaggle-ml-base-layer` stack (`kbench → kaggle → metis → ariadne`). It owns the
reproducible unit of ML work (the **experiment**) and, as they land, the step-runner
and the Dataset/Split/step-type data plane. "Platform-independent" test: *would this be
identical on a non-Kaggle platform?* — if yes, it lives here.

- [experiment datatype](experiment.md) — the reproducible pipeline noun: the CUE schema
  (`#Experiment`/`#Step`/`#Status`/`#Run`), the `xx-datatype` authoring prototype, the
  `vocabulary validate-instance` structural validator, and the enforcement merge-check.
  (metis#1 M1.)
- [workflow/](workflow) — inherited ariadne workflow docs (symlink into the substrate).

**Roadmap (metis#1):** M1 (experiment datatype) + M2 (Go step-runner: `cmd/metis run` +
pure `pkg/experiment` `Parse`/`Validate`/`TopoSort`, semantics enforced on read, steps run
as subprocesses over a `steps/<layer>/<steptype>` + files contract) **shipped**. **M3** adds
the Python data plane (Dataset/Schema/Split + `cv-split`/`train`/`predict` step-types
conforming to M2's step contract). The end-to-end proof is kbench's Titanic walking skeleton
(kbench#1).

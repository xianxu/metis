"""Schema — the pure column-role model of a Dataset (metis#1 M3).

A Schema names each column's ROLE (id / feature / target / weight) and its dtype.
It is the Python mirror of the experiment's data model: the split/train/predict
steps ask the schema which columns are features and which is the target rather
than hard-coding names, so the same step-types work on any tabular dataset
(ARCH-DRY — one place decides roles). Pure: no IO (serialization lives in
metis.io).
"""

from __future__ import annotations

from dataclasses import dataclass

# The closed set of column roles. `feature` columns are model inputs; the single
# `target` is the label; `id` identifies a row (carried into predictions, never a
# model input); `weight` is an optional per-row sample weight; `source` is a raw
# column carried through for feature-engineering steps that know it — never a
# model input, and (unlike feature/target) it may hold strings/NaN (metis#35).
ROLES = frozenset({"id", "feature", "target", "weight", "source"})


@dataclass(frozen=True)
class Schema:
    columns: dict[str, str]  # column name -> role (one of ROLES)
    dtypes: dict[str, str]   # column name -> pandas dtype string (e.g. "float64")

    def __post_init__(self) -> None:
        bad = {name: role for name, role in self.columns.items() if role not in ROLES}
        if bad:
            raise ValueError(f"unknown column role(s): {bad}; roles must be one of {sorted(ROLES)}")
        targets = self._by_role("target")
        if len(targets) > 1:
            raise ValueError(f"a Schema has at most one target column; got {targets}")

    def _by_role(self, role: str) -> list[str]:
        # Insertion order preserved (dict is ordered) — deterministic feature order.
        return [name for name, r in self.columns.items() if r == role]

    def feature_cols(self) -> list[str]:
        return self._by_role("feature")

    def target_col(self) -> str | None:
        targets = self._by_role("target")
        return targets[0] if targets else None

    def id_col(self) -> str | None:
        ids = self._by_role("id")
        return ids[0] if ids else None

    def weight_col(self) -> str | None:
        weights = self._by_role("weight")
        return weights[0] if weights else None

    @classmethod
    def from_dict(cls, d: dict) -> "Schema":
        return cls(columns=dict(d["columns"]), dtypes=dict(d.get("dtypes", {})))

    def to_dict(self) -> dict:
        return {"columns": dict(self.columns), "dtypes": dict(self.dtypes)}

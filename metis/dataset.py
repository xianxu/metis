"""Dataset — the canonical in-memory form the data plane operates on (metis#1 M3).

A Dataset bundles a Schema with train / optional test DataFrames plus free
provenance metadata. It is the modality-agnostic envelope every Adapter converts
raw data into (tabular loaders now; other modalities later) and the form the
cv-split / train / predict step-types consume. Pure data structure: load/save is
the thin metis.io layer (ARCH-PURE).
"""

from __future__ import annotations

from dataclasses import dataclass, field

import pandas as pd

from metis.schema import Schema


@dataclass
class Dataset:
    schema: Schema
    train: pd.DataFrame
    test: pd.DataFrame | None = None
    provenance: dict = field(default_factory=dict)

    def X(self, df: pd.DataFrame) -> pd.DataFrame:
        """The feature matrix for df, in the schema's feature order (pure selector)."""
        return df[self.schema.feature_cols()]

    def y(self, df: pd.DataFrame) -> pd.Series:
        """The target vector for df. Raises if the schema declares no target."""
        target = self.schema.target_col()
        if target is None:
            raise ValueError("dataset schema declares no target column")
        return df[target]

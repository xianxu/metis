"""Pure tests for metis.encode.cross_fit_target_encode (no IO)."""
import numpy as np
import pandas as pd
import pytest

from metis.encode import cross_fit_target_encode


def _naive_incl_self(groups, y):
    """The LEAKY baseline: group mean INCLUDING the row itself (what #20 must beat)."""
    s = pd.Series(y, dtype=float)
    return s.groupby(np.asarray(groups)).transform("mean").to_numpy()


def _corr(a, b):
    return float(np.corrcoef(a, b)[0, 1])


def test_kfold_no_self_leak_on_random_data():
    """y independent of group ⇒ NO real signal. Naive (incl-self) encoding correlates
    with the row's own label (leak); cross-fit does not."""
    rng = np.random.default_rng(0)
    groups = np.repeat(np.arange(100), 2)          # 100 groups of size 2
    rng.shuffle(groups)
    y = rng.integers(0, 2, size=len(groups))
    naive = _naive_incl_self(groups, y)
    enc = cross_fit_target_encode(groups, y, strategy="kfold", n_folds=5, m=10.0, seed=0)
    assert abs(_corr(naive, y)) > 0.4              # the leak the naive path introduces
    assert abs(_corr(enc, y)) < 0.15               # cross-fit removes it


def test_kfold_preserves_real_between_group_signal():
    """When y genuinely tracks the group, cross-fit must RECOVER it (not null everything)."""
    rng = np.random.default_rng(1)
    rates = {0: 0.1, 1: 0.4, 2: 0.6, 3: 0.9}
    groups = np.repeat(np.arange(4), 50)               # 4 groups of size 50 (large ⇒ stable OOF)
    y = np.array([rng.random() < rates[g] for g in groups], dtype=float)
    enc = cross_fit_target_encode(groups, y, strategy="kfold", n_folds=5, m=10.0, seed=0)
    assert _corr(enc, y) > 0.3                          # recovers the legitimate signal
    assert enc.std() > 0.1                              # NOT a constant (kills the return-prior cheat)
    for g, p in rates.items():                          # each group's enc ≈ its true rate
        assert abs(enc[groups == g].mean() - p) < 0.15


def test_non_fit_rows_get_full_fit_and_unseen_gets_prior():
    groups = np.array(["a", "a", "a", "b", "b", "zzz"])
    y      = np.array([1.0, 1.0, 0.0, 0.0, 0.0, np.nan])   # last row = non-fit, unseen group
    fit    = np.array([True, True, True, True, True, False])
    enc = cross_fit_target_encode(groups, y, fit_mask=fit, m=0.0, seed=0)   # m=0 ⇒ raw means
    # non-fit 'zzz' never seen among fit rows ⇒ prior = mean(fit y) = 2/5 = 0.4
    assert enc[5] == pytest.approx(0.4)


def test_non_fit_known_group_gets_full_fit_mean():
    groups = np.array(["a", "a", "a", "a"])
    y      = np.array([1.0, 1.0, 0.0, np.nan])
    fit    = np.array([True, True, True, False])
    enc = cross_fit_target_encode(groups, y, fit_mask=fit, m=0.0, seed=0)
    assert enc[3] == pytest.approx(2 / 3)              # full-fit mean of group 'a' over fit rows


def test_deterministic_under_seed():
    rng = np.random.default_rng(2)
    groups = rng.integers(0, 30, size=120)
    y = rng.integers(0, 2, size=120)
    a = cross_fit_target_encode(groups, y, seed=7)
    b = cross_fit_target_encode(groups, y, seed=7)
    assert np.array_equal(a, b)


def test_ship_path_all_fit_no_crash():
    rng = np.random.default_rng(3)
    groups = rng.integers(0, 20, size=80)
    y = rng.integers(0, 2, size=80)
    enc = cross_fit_target_encode(groups, y, fit_mask=None, seed=0)   # all rows fit
    assert enc.shape == (80,) and np.all(np.isfinite(enc))


def test_loo_within_group_structure_is_label_invertible():
    """LOO's residual leak — the reason kfold is the default. Within a REALIZED group, raw-LOO
    encoding enc_i = (S - y_i)/(n-1) is a deterministic function of (group, own label): all
    survivors collapse to one value, all non-survivors to another, separated by exactly 1/(n-1).
    A flexible model that isolates the group can invert this to recover the label. Seeded noise
    (loo_noise>0) blurs it; kfold has no such deterministic per-label structure (Step 1 proves
    kfold doesn't leak). This is NOT visible in marginal corr(enc, y) — it is a within-group
    property — so we assert the structure directly, not a fragile correlation inequality."""
    groups = np.array(["g", "g", "g", "g"])        # one size-4 group, labels [1,1,0,0], S=2
    y = np.array([1.0, 1.0, 0.0, 0.0])
    enc = cross_fit_target_encode(groups, y, strategy="loo", loo_noise=0.0, m=0.0, seed=0)
    assert enc[0] == pytest.approx(1 / 3) and enc[1] == pytest.approx(1 / 3)   # survivors collapse
    assert enc[2] == pytest.approx(2 / 3) and enc[3] == pytest.approx(2 / 3)   # non-survivors collapse
    assert enc[2] - enc[0] == pytest.approx(1 / 3)                             # the 1/(n-1) gap


def test_loo_deterministic_and_finite_with_noise():
    """LOO with seeded additive noise: finite + reproducible under seed (the safe LOO form)."""
    rng = np.random.default_rng(4)
    groups = rng.integers(0, 30, size=120)
    y = rng.integers(0, 2, size=120)
    a = cross_fit_target_encode(groups, y, strategy="loo", loo_noise=0.1, m=1.0, seed=5)
    b = cross_fit_target_encode(groups, y, strategy="loo", loo_noise=0.1, m=1.0, seed=5)
    assert np.all(np.isfinite(a)) and np.array_equal(a, b)


def test_unknown_strategy_rejected():
    with pytest.raises(ValueError, match="unknown strategy"):
        cross_fit_target_encode(np.array([1, 1]), np.array([0.0, 1.0]), strategy="bogus")

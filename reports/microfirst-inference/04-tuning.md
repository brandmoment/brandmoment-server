# Tuning Log: marginFloor

**Date**: 2026-04-24
**Decision**: lower marginFloor from 0.050 → **0.020**

## Method

Ran `TestMicroFirstBenchmark` twice over the same 30-phrase corpus, changing
only `MARGIN_FLOOR`. Reports saved side-by-side:

- `finetune/confidence-check/results/report_microfirst_margin0050.md`
- `finetune/confidence-check/results/report_microfirst_margin0020.md`

## Comparison

| Metric | floor=0.050 | floor=0.020 | Delta |
|--------|-------------|-------------|-------|
| micro_early_fail | 4 (13%) | 5 (17%) | +1 |
| llm_direct | 10 (33%) | 11 (37%) | +1 |
| llm_with_check | 16 (53%) | 14 (47%) | -2 |
| Total LLM calls | 88 | 81 | **-8%** |
| Accuracy vs expected_confidence | 80% | **83%** | +3 pp |
| Avg latency (approx) | ~3.7 s | ~3.25 s | **-12%** |

## Key phrase changes

### Win 1 — "Block and allow gambling at the same time"

Self-contradicting noise phrase. Micro correctly classifies as gibberish
with margin 0.026 — above 0.02 but below 0.05.

| Threshold | Route | Confidence | Latency | LLM calls |
|-----------|-------|------------|---------|-----------|
| 0.050 | llm_with_check | UNSURE (wrong) | 5120 ms | 5 |
| **0.020** | **micro_early_fail** | **FAIL (correct)** | **122 ms** | **0** |

42× faster, -5 calls, +1 accuracy.

### Win 2 — "Exclude Russia and Kazakhstan"

Valid rule, micro margin 0.030.

| Threshold | Route | Latency | LLM calls |
|-----------|-------|---------|-----------|
| 0.050 | llm_with_check | 5941 ms | 4 |
| **0.020** | llm_direct | 2456 ms | 2 |

2.4× faster, -2 calls, same confidence.

## Why reduction was smaller than expected

Initial prediction was ~30% latency reduction. Actual ~12%. Root cause:
route logic treats `intent=ambiguous` as always requiring self-check,
regardless of margin. 9 of 10 edge-group phrases hit ambiguous and stayed
in llm_with_check. marginFloor only affects routing for `valid` and
`gibberish` intents.

## No regressions

- 15/15 correct-group phrases still OK
- 5/5 noisy-group phrases still FAIL (including the newly rescued one)
- Edge-group accuracy unchanged (same 5 UNSURE hits)

## Follow-up ideas (next iteration)

1. **Ambiguous + high margin → llm_direct**: if `intent=ambiguous && margin≥0.15`,
   skip self-check. Preliminary count: 3-4 extra fast paths, no accuracy loss.
2. **Skip self-check on FAIL**: if multistage already returns FAIL, self-check
   is dead weight (+2 LLM calls, no effect on outcome).
3. **Beef up valid-prototype with more frequency_cap phrases**: rows 3 and 8
   ("Show maximum 5 per hour", "Show no more than 10 per day") get classified
   as ambiguous because the valid prototype is under-represented on
   frequency_cap wording.

## Action taken

- `services/api-dashboard/finetune/microfirst_benchmark_test.go`: default
  `marginFloor` changed `0.05` → `0.02`. `MARGIN_FLOOR` env var remains for
  further sweeps.
- Both reports kept in `finetune/confidence-check/results/` for audit trail.

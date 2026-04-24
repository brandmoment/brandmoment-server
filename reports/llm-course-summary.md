# LLM Course — Summary Across All Days

**Domain**: NL phrase → PublisherRule JSON (ad-network rule engine)
**Corpus**: 30 phrases (15 correct, 10 edge, 5 noisy)
**Date range**: 2026-04-21 → 2026-04-24

## Timeline

| Day | Task | Approach | Result |
|-----|------|----------|--------|
| Day 6 | Confidence estimation | constraint + self_check (baseline) | gpt-4o-mini 77% accuracy |
| Day 6 | Multi-provider | OpenAI vs Gemini | gpt-4o-mini >> gemini |
| Day 8 | Model routing | gpt-4o-mini → gpt-4o fallback on low confidence | 1/11 escalations helped, expensive |
| Day 9 | Multi-stage inference | analyze → extract → assemble | +3 correct (22 → 25 OK), 2× latency |
| Day 10 | Micro-model first | embedding gate → multi-stage + self-check | 83% accuracy, 5/5 noisy early-fail |

---

## Day 6 — Confidence Estimation

**What**: baseline NL → JSON parser with two confidence approaches (constraint validation, self-check via LLM verify).
**Models compared**: `openai/gpt-4o-mini`, `gemini/gemini-2.0-flash`, `gemini/gemini-2.5-flash`.

| Provider | Accuracy (30) | Constraint | Self-check |
|----------|---------------|------------|------------|
| openai/gpt-4o-mini | **23 / 77%** | 22 OK + 8 FAIL | 19 OK + 11 UNSURE |
| gemini/gemini-2.5-flash | 7 / 23% | 2 OK + 28 FAIL | 2 OK + 1 UNSURE + 27 FAIL |
| gemini/gemini-2.0-flash | 5 / 17% | 0 OK + 30 FAIL | 0 OK + 30 FAIL |

**Takeaway**: OpenAI dominates on this structured-output task. Gemini fails parsing entirely (likely prompt sensitivity / markdown fences). Default to gpt-4o-mini for rest of experiments.

**Lesson**: constraint validation (schema check) is free and catches hard errors. Self-check adds UNSURE middle tier but doubles token cost.

---

## Day 8 — Model Routing

**What**: try cheap model first (gpt-4o-mini); if confidence low, fall back to strong model (gpt-4o).

| Metric | Value |
|--------|-------|
| Stayed on primary | 19 / 30 (63%) |
| Escalated to fallback | 11 / 30 (37%) |
| Total input tokens | 20,587 |
| Total output tokens | 2,062 |
| Accuracy rescues | 1 / 11 (row 24 UNSURE → OK) |

**Takeaway**: fallback is mostly wasted. 10 / 11 escalations stayed FAIL — gpt-4o cannot recover gibberish or deeply ambiguous phrases that are outside the rule schema. The one rescue (`Allow only mobile`) was borderline.

**Lesson**: escalation policy needs to be confidence-conditional (UNSURE → escalate; FAIL → don't bother). Bigger model ≠ better when the input is genuinely unparseable.

---

## Day 9 — Multi-Stage Inference

**What**: decompose monolithic parse into 3 stages — Stage 1 analyze (count + types), Stage 2 extract (config per type, one LLM call each), Stage 3 assemble + constraint check (pure Go).

| Metric | Monolithic | Multi-Stage |
|--------|-----------|-------------|
| OK | 22 | **25** |
| UNSURE | 0 | 0 |
| FAIL | 8 | **5** |
| Avg latency | 1.08 s | 2.04 s |
| Total tokens | 12,525 | 12,374 |

**Wins (FAIL → OK)**: 3 phrases — `Block competitors`, `Limit frequency but only on weekends`, `Block and allow gambling at the same time`.
**Losses**: 0.

**Takeaway**: decomposition helps on ambiguous / compound phrases. Token cost is roughly identical (shorter prompts × more calls). Latency doubles.

**Lesson**: specializing each stage's prompt for a narrower task ("identify types only") produces more reliable JSON than asking the LLM to do everything at once.

---

## Day 10 — Micro-Model First

**What**: embedding-based classifier (`text-embedding-3-small`) gates the multi-stage pipeline. 3 intents (valid / ambiguous / gibberish). 3 routes:

- `micro_early_fail` — gibberish with high margin → no LLM call
- `llm_direct` — valid with high margin → normal multi-stage
- `llm_with_check` — ambiguous or low margin → multi-stage + self_check

**Tuning sweep on marginFloor**:

| Metric | floor=0.050 | floor=0.020 (final) |
|--------|-------------|---------------------|
| micro_early_fail | 4 (13%) | 5 (17%) |
| llm_direct | 10 (33%) | 11 (37%) |
| llm_with_check | 16 (53%) | 14 (47%) |
| Total LLM calls | 88 | **81** |
| Avg latency | ~3.7 s | **~3.25 s** |
| Accuracy | 80% | **83%** |

**Win**: `Block and allow gambling at the same time` (micro margin 0.026) now early-fails at 122 ms instead of 5120 ms, correct FAIL instead of wrong UNSURE.

**Takeaway**: cheap semantic gate catches clear-noise phrases without an LLM call. But self_check on ambiguous phrases dominates total latency.

**Lesson**: embedding + cosine + margin is a ~$0 pre-filter that works. The bottleneck moved from "is the LLM wrong?" to "is self-check worth its 2 extra calls?".

---

## Cross-Day Comparison

### Accuracy evolution (all against gpt-4o-mini backbone)

```
Day 6 (constraint)          22/30   73%   ████████████████████░░░░░░
Day 6 (self_check)          19/30   63%   █████████████████░░░░░░░░░
Day 9 (monolithic)          22/30   73%   ████████████████████░░░░░░
Day 9 (multi-stage)         25/30   83%   ██████████████████████░░░░
Day 10 (micro + mult + sc)  25/30   83%   ██████████████████████░░░░
```

(“Accuracy” here = confidence status matches `expected_confidence` in the corpus, not just OK count.)

### Cost / call profile

| Day | LLM calls / phrase (avg) | Extra API calls |
|-----|-------------------------|-----------------|
| Day 6 constraint | 1 | 0 |
| Day 6 self_check | 2 | 0 |
| Day 8 routing (stayed) | 2 | 0 |
| Day 8 routing (escalated) | 4 | 0 |
| Day 9 multi-stage | 2–4 (1 analyze + N extract) | 0 |
| Day 10 direct | 2–4 | 1 embedding |
| Day 10 with check | 4–6 | 1 embedding |
| Day 10 early fail | **0** | 1 embedding |

### Latency

| Day | Avg latency |
|-----|-------------|
| Day 6 constraint | ~1.0 s |
| Day 6 self_check | ~2.0 s |
| Day 8 routing | ~3.1 s |
| Day 9 multi-stage | ~2.0 s |
| Day 10 (floor 0.05) | ~3.7 s |
| Day 10 (floor 0.02) | ~3.25 s |

Latency grew until Day 10 embedding routing started paying off on noisy phrases.

### Noisy group (5 gibberish phrases) — latency per phrase

| Day | kakashki | xyzzy | Show ads | Do nothing | Block+allow |
|-----|----------|-------|----------|------------|-------------|
| Day 9 baseline | ~760 ms (1 LLM) | ~820 | ~720 | ~760 | ~1230 |
| Day 10 two-level | **114 ms (0 LLM)** | **195** | **120** | **107** | **122** |

**6–10× faster on pure noise with 0 LLM calls** — the flagship win of micro-first.

---

## What each day taught

| Day | Core lesson |
|-----|-------------|
| 6 | Confidence tier (OK / UNSURE / FAIL) is more useful than binary correct/incorrect. Model choice matters more than prompt tricks for structured output. |
| 8 | Big-model fallback is mostly wasted; escalation must gate on confidence *tier*, not just "we're unsure, try again". |
| 9 | Decomposition > monolithic. Narrower prompts produce cleaner JSON. Cost is latency, not tokens. |
| 10 | Cheap pre-filters (embedding) cut out the easy rejects before the expensive path. Tuning thresholds visibly affects cost without touching accuracy. |

---

## Where it goes next (follow-ups across all days)

1. **Ambiguous + high-margin → llm_direct** (skip self_check). Likely 10–15% more latency saving with no accuracy loss.
2. **Skip self_check on multistage FAIL** — pure dead weight (+2 LLM calls for no effect).
3. **Corpus expansion**: frequency_cap and multi-rule phrases are under-represented → misclassified as ambiguous. Add ~20 phrases, rebuild prototypes.
4. **Cache embeddings** on phrase hash — repeat queries become free after first call.
5. **Combine routing + micro**: if two-level sends to llm_with_check AND confidence still low, *then* escalate to gpt-4o (instead of always).
6. **Dashboard UI** for the parse endpoint (handler exists, UI deferred).

---

## Reports index

| Day | File |
|-----|------|
| Day 6 | `finetune/confidence-check/results/report_openai_gpt-4o-mini.md` |
| Day 6 | `finetune/confidence-check/results/report_gemini_gemini-2.0-flash.md` |
| Day 6 | `finetune/confidence-check/results/report_gemini_gemini-2.5-flash.md` |
| Day 6 | `reports/confidence-check-rule-parser/05-report.md` (task report) |
| Day 8 | `finetune/confidence-check/results/report_routing.md` |
| Day 8 | `reports/model-routing-feature/` |
| Day 9 | `finetune/confidence-check/results/report_multistage.md` |
| Day 9 | `reports/multi-stage-inference/` |
| Day 10 | `finetune/confidence-check/results/report_microfirst_margin0050.md` |
| Day 10 | `finetune/confidence-check/results/report_microfirst_margin0020.md` |
| Day 10 | `reports/microfirst-inference/` (spec + impl + report + tuning) |

# Task: Micro-First Enum Classifier (Variant A2)

Profile: Feature  |  Stage: Done  |  Report: 03-report.md

Created: 2026-04-24  |  Updated: 2026-04-25

## Context

Day 10 first pass (micro gate → full multistage) has spec misalignment: spec
requires LLM only on UNSURE/low-confidence, but current impl calls multistage
on valid phrases too (llm_direct). Only 17% handled without LLM.

Variant A2: expand Intent taxonomy from 3 → 7 classes (5 rule types +
ambiguous + invalid). For high-margin rule-type predictions, skip the analyze
stage of multi-stage and call extract-only (1 LLM call instead of 2+).

Goal: majority of correct-group phrases routed to 1-LLM-call path.

## Progress

- [x] 01-spec.md — enum taxonomy + routes + acceptance criteria
- [x] 02-implement-go.md — 7 files modified
- [x] 03-validate.md — go build + vet + llm/ tests all green
- [x] 03-report.md — comparison table filled with live results
- [x] Live benchmark — `report_microfirst_a2_margin0020_rtf0050.md`

## Result summary

- 15/15 correct → `micro_answer` (1 LLM call, ~1.1s avg)
- 4/5 noisy → `micro_early_fail` (0 LLM, ~166ms)
- 10/10 edge → `llm_with_check` (unchanged from v1)
- 1 noisy regression: row 27 "Block and allow gambling..." (blocklist top-1
  due to word overlap, fixable with contradiction prototype)
- **63% handled without full pipeline** (target ≥50%) — spec satisfied

## Follow-ups (next iteration, optional)

- Add contradiction phrases to invalid prototype to recover row 27
- Wire `TwoLevelParser` into HTTP handler
- Skip self_check on multistage FAIL

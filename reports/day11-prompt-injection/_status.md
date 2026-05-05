# Task: Day 11 — switch production rule parser to HardenedSystemPrompt
Profile: Feature  |  Stage: Done  |  Next: —
Created: 2026-04-30  |  Updated: 2026-05-03

## Context

Day 11 LLM-security course task. Fully migrated production prompts to hardened
versions (SEC-1..SEC-7) across all LLM callsites. Deleted orphaned finetune
artifact. All 7 prompt-injection attacks HELD against production prompt.

## Completed

1. `multistage.go` — both stage1 and stage2 system prompts hardened with
   SEC-1..SEC-7 adapted to their output schemas. Both callsites now use
   `WrapPhrase`.
2. `finetune/hardened_prompt.go` — deleted (superseded by `llm.SystemPrompt`).
3. `finetune/promptinjection_attack_test.go` — simplified: always uses
   `llm.SystemPrompt` + `llm.WrapPhrase`, single report path, dual leak signals
   for A3/A5.

## Report

`reports/day11-prompt-injection/03-implement-hardened.md`

# Task: Confidence Estimation for LLM-based Rule Parser

Profile: Feature  |  Stage: Done  |  Report: 05-report.md

Created: 2026-04-21  |  Updated: 2026-04-21 20:00

## Context

Учебное задание с курса LLM: «Оценка уверенности и контроль качества инференса». NL-фраза → PublisherRule JSON с confidence. 2 подхода: constraint, self-check. 2 провайдера: OpenAI (gpt-4o-mini) + Gemini (gemini-2.0-flash).

## Completion Summary

✓ Spec → Implement → Test → Validate → Report

**Deliverables**:
- Go backend: `internal/llm/` package (client, openai, gemini, constraint, self_check)
- Service: `internal/service/rule_parser.go` (orchestration)
- Handler: `internal/handler/rule_parser.go` (HTTP POST /v1/publisher-rules/parse)
- Router: Registered endpoint (editor+ RBAC)
- Config: OpenAIAPIKey + GeminiAPIKey from env
- Tests: 76 new test cases (constraint 33, self_check 15, service 20, handler 8) — all passing
- Benchmark: 30-phrase corpus + test harness
- TypeScript: Types + component placeholders (UI build deferred)

**Validation**:
- Go: build ✓, vet ✓, test ✓ (511 tests)
- TypeScript: 4 pre-existing auth type errors (unrelated to feature)

**Status**: Feature-complete. Ready for real API key testing + dashboard UI build.

## Next Steps (Out of Scope)

1. Run benchmark with live OpenAI/Gemini keys → populate results/report_*.md
2. Build dashboard UI page
3. Fix auth type errors in lib/auth-client.ts
4. E2E integration tests
5. Security review (auth/multi-tenancy)

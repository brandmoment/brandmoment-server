# Peer Repos Review: Agent Patterns & Best Practices
Date: 2026-04-18
Repos analyzed: 10

## TL;DR — Top Findings

1. **vaporphd/scrumban** — лучшая оркестрация: автономный loop (implement → smoke → review → auto-merge), `## Handoff` блоки для маршрутизации между агентами, `followup.md` как cross-session memory
2. **Aristman/my_pi** — самый продвинутый runtime: worker pool с DAG зависимостей, VERDICT протокол, RAG memory с confidence decay
3. **nikitaluga/AIChallenge** — `/spec-interview` → `.specs/` → auto-build pipeline, memory/ directory
4. **kuzminolegnik/GAME-CARD** — browser smoke testing через agent-browser MCP, "право не согласиться" директива
5. **flying-develop/ai-advanced** — тестовые задачи встроены в профили для валидации промптов
6. **ivanlardis/aI-advent-challenge** — tool scoping per agent, severity system (BLOCKER/IMPORTANT/NIT)

---

## Detailed Analysis by Repo

### 1. vaporphd/scrumban — THE STANDOUT

8 агентов: architect, bug-hunter, ci-devops, docs-writer, explorer, implementer, reviewer, smoke-tester.

**Автономный pre-merge loop (zero user interaction):**
```
implementer → smoke-tester → reviewer → auto-merge
                  |                |
                  v                v
            implementer       implementer
            (fix regression)  (address findings)
```
User input per PR: 1 слово (issue number / "go"). Loop завершается только на auto-merge или unresolvable blocker.

**`## Handoff` блоки для маршрутизации:**
```markdown
## Handoff
next: smoke-tester (frontend / user-visible feature — fresh PR)
  | next: reviewer (pure backend / infra / docs — fresh PR)
  | next: implementer (changes-requested — findings posted to issue #N)
  | next: human (blocker — pre-existing fail / unresolvable dep / scope question)
```
Main читает этот блок и определяет следующего агента. Это **протокол межагентной коммуникации** — структурированный текст, не файлы.

**followup.md как cross-session memory:**
- Два раздела: Status и Next
- Полностью **перезаписывается** при каждом merge (не append)
- Reviewer **блокирует merge** если followup.md отсутствует или устарел
- "git log is the history, not this file"

**Strict agent boundaries:**
- architect: НЕ пишет код в `backend/app/` или `frontend/src/`
- explorer: НЕ использует Edit или Write
- reviewer: НЕ пушит коммиты
- implementer: НЕ мержит PR (только main session)

**Nits-as-bugs:** Все findings (must-fix, should-fix, nit) блокируют merge. `approve-with-suggestions` deprecated.

**ADR gate:** Architect пишет ADR с обязательной секцией Consequences. Новые подсистемы требуют ADR до merge.

---

### 2. Aristman/my_pi

Кастомный runtime на pi-coding-agent (не Claude Code). 4 типа агентов: explore, plan, implement, verify.

**Worker pool с DAG зависимостей:**
- 1724 строк TypeScript-оркестратора
- Write lock — только один implement worker; read-only workers параллельно
- JSONL RPC протокол между workers
- Tasks с `blocks[]` / `blockedBy[]` для tracking зависимостей

**VERDICT протокол:** Verify agent возвращает `VERDICT: PASS/FAIL/PARTIAL` — coordinator парсит и маршрутизирует.

**RAG memory с confidence decay:**
- Vector store с ANN индексом
- Auto-extraction memories из разговоров
- Confidence decay: 30d → 0.85, 90d → 0.7, 180d → 0.5
- Deduplication + topic-shift detection через cosine similarity

**Model routing:** Лёгкие модели для explore, тяжёлые для implement. Автоматический fallback cloud ↔ local (Ollama).

**Workspace staging:** Артефакты создаются в `workspace/`, тестируются, затем deploy. Rollback при ошибке.

---

### 3. nikitaluga/AIChallenge

Kotlin Multiplatform (Android/iOS/Server). 3 агента + 1 slash command.

**`/spec-interview` → `.specs/` → auto-build pipeline:**
1. Codebase analysis
2. Deep interview через AskUserQuestion
3. Spec generation в `.specs/` (Kotlin code, file paths, MVI contracts, ASCII UI layouts)
4. Auto-launch 6+ parallel agents для реализации

`.specs/` — промежуточные артефакты, мост между interview и implementation.

**`memory/` directory:** Файлы `project_day*.md` с YAML frontmatter — persistent knowledge base между сессиями.

**Agent efficiency guidelines:**
- "Known file paths → use Read in parallel, never spawn Task agent"
- "Batch all edits into one message"
- "Answer with code, not words. No trailing summary. The user sees the diff."

---

### 4. kuzminolegnik/GAME-CARD-NUCDESERT (GitLab)

Node.js/TypeScript card game. 4 slash commands: bug-fix, research, review, smoke.

**Browser smoke testing через agent-browser MCP:**
- `smoke.md` спавнит sub-agent
- Sub-agent читает `tests/smoke/scenarios.md` (human-readable spec)
- Выполняет через `npx agent-browser open/snapshot/screenshot/close`
- Сохраняет screenshots + `report.md` в `tests/smoke/result/<testcase>/<datetime>/`
- Реальные результаты в репо (screenshots от April 17, 2026)

**Conditional DOCS stage:** `PLANNING → EXECUTION → VALIDATION → DONE → DOCS`. DOCS только для архитектурных изменений — не для каждого PR.

**"Right to disagree":** Agent MUST object если решение ведёт к хакам, security holes, или tech debt. "Silent agreement with a bad decision = error."

**Model + effort annotations:** `model: claude-opus-4-7, effort: high` на командах.

---

### 5. flying-develop/ai-advanced

Python Telegram bot. 3 профиля в `.claude/profiles/`: bugfix, migration, research.

**Migration profile:** Dedicated workflow для Alembic:
```
UNDERSTAND → MODIFY MODEL → GENERATE MIGRATION → VERIFY → REPORT
```
Verify = round-trip test: `upgrade head → downgrade -1 → upgrade head`.

**Test tasks встроены в профили:** Каждый профиль содержит конкретные тестовые баги/вопросы для самопроверки. Meta-checklist: "Агент нашёл корневую причину, а не симптом?"

**State machine:** `IDLE → ANALYSIS → PLANNING → EXECUTION → VALIDATION → DONE` с правилами переходов.

---

### 6. ivanlardis/aI-advent-challenge

Python AI assistant (Chainlit + LangChain). 3 агента + test-full-cycle skill.

**Tool scoping per agent:**
- bug-fix: `Bash, Glob, Grep, Read, Edit, Write` — full access
- research: `Glob, Grep, Read` — read-only, даже Bash запрещён
- review: `Bash, Glob, Grep, Read` — read + git diff, no write

**Severity system:** `BLOCKER | IMPORTANT | NIT`. Final verdict: "N blocker / M important / K nit. Can we merge?"

**test-full-cycle skill:** pytest → start server → Playwright smoke → stop server → report → diagnose failures. Всё в одном skill.

**Anti-hallucination:** "Don't invent API", "Don't soften tests — fix the code not the test."

---

### 7. AndVl1/AdventProject

Kotlin Multiplatform. 4 inline профиля в CLAUDE.md.

**Refactor profile:** "Рефакторинг без изменения семантики. Любое изменение поведения — повод остановиться и спросить."

**Feature profile:** Bottom-up order: domain → data → presentation. UI — последний шаг.

**"Что НЕ делал (осознанно)"** — секция в отчётах для явной документации out-of-scope решений.

---

### 8-10. max1ly/ai_advent, KonstantinKhan/atlas-project, RomAn-8/guardsar_claude

**max1ly/ai_advent:** Нет agent infrastructure. `docs/plans/` с ручными design docs.

**KonstantinKhan/atlas-project:** Docs-as-navigation: 3-level hierarchy (OVERVIEW → INDEX → details). "If docs contradict code — that is a docs bug."

**RomAn-8/guardsar_claude:** Tiered test commands: Level 1 (unit tests) и Level 2 (smoke via localhost). Permission scoping в `settings.local.json`.

---

## What We Have That Others Don't

| Our advantage | Details |
|---|---|
| Multi-profile system with strict transitions | Bug Fix / Research / Update Docs with allowed/forbidden state transitions |
| Agent-per-stage mapping table | Explicit table of which agents run at which stage |
| Mandatory agent launch policy | Enforced parallel agent execution, no shortcuts |
| Dedicated report-writer agent | Others write reports inline |
| 14 specialized subagents | Most repos have 3-4 |

## What We're Missing

| Pattern | Source | Priority | What it gives us |
|---|---|---|---|
| `## Handoff` block protocol | scrumban | **HIGH** | Structured inter-agent routing without main pересказ |
| `followup.md` cross-session memory | scrumban | **HIGH** | Persistent context between conversations |
| Tool scoping per agent | aI-advent-challenge | **HIGH** | Prevent accidental writes in read-only agents |
| Autonomous loop (implement → test → review → merge) | scrumban | **HIGH** | Zero-question PR flow |
| Feature profile | AdventProject | **HIGH** | No profile for writing new code — main daily work |
| `/spec-interview` → `.specs/` → auto-build | AIChallenge | **MEDIUM** | Chain interview into implementation |
| Smoke testing with browser agent + scenarios | GAME-CARD | **MEDIUM** | Agent-driven E2E testing |
| Test tasks embedded in profiles | ai-advanced | **MEDIUM** | Self-validation of profile quality |
| Severity system (BLOCKER/IMPORTANT/NIT) | aI-advent-challenge | **MEDIUM** | Structured review output |
| "Right to disagree" directive | GAME-CARD | **MEDIUM** | Prevent silent compliance with bad decisions |
| Autonomy levels (auto vs supervised) | scrumban | **MEDIUM** | User chooses how much agent asks |
| Conditional DOCS stage | GAME-CARD | **LOW** | Don't over-document trivial changes |
| VERDICT protocol (PASS/FAIL/PARTIAL) | my_pi | **LOW** | Machine-parseable agent output |
| Confidence decay in memory | my_pi | **LOW** | Auto-deprecate stale memories |
| Agent efficiency guidelines | AIChallenge | **LOW** | "Batch edits, parallel reads, no trailing summary" |
| "Что НЕ делал (осознанно)" report section | AdventProject | **LOW** | Document conscious out-of-scope decisions |
| Anti-hallucination rules | aI-advent-challenge | **LOW** | "Fix code not test", "don't invent API" |

---

## Design Decisions from This Session

Based on peer review analysis + our discussion, key decisions for next iteration:

### 1. Feature Profile (NEW)
```
Spec → Implement → Test → Validate → Report → Done
```
- Spec: main or /interview
- Implement: go-builder / ts-builder / sql-builder (parallel by layer)
- Test: go-test-writer / e2e-test-writer (write tests for new code)
- Validate: test-runner (run everything)
- Report: report-writer

### 2. New Test Agents
- `go-test-writer` — finds uncovered Go modules, writes tests
- `e2e-test-writer` — writes Playwright E2E scenarios from descriptions
- `test-runner` — stays unified, runs Go + TS + E2E

### 3. Autonomy Levels
Default: **auto** — agent makes decisions, escalates only on blockers.
User override: "ask me before fixing" → supervised mode.

### 4. Worktree Rules
- Parallel tasks → different worktrees (ok)
- Agents within same task → one worktree (mandatory)
- No `isolation: "worktree"` for agents inside a loop

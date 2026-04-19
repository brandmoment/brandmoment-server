# Local LLM Models Review

## Setup
- IDE: VS Code + Continue.dev → Cline
- Runtime: Ollama
- Hardware: MacBook Pro, 36 GB RAM, Apple Silicon
- Project: BrandMoment Server (Go + Next.js monorepo)

## Task
Organizations CRUD: миграция, sqlc queries, repository, service, handler.
Та же задача что была выполнена Claude Code в Day 1 (feature/rules-claude-v2).

## System Prompt
Сокращённый CLAUDE.md → `.continuerc.yaml` (systemMessage). Убраны профили, агенты, валидация — оставлены: стек, слои, naming, multi-tenancy, sqlc конвенции.

---

## Models Tested

### 1. DeepSeek Coder V2 16B
- Size: 8.9 GB
- Result: **не запустился в agent mode**
- Error: `does not support tools`
- Verdict: не поддерживает tool calling, бесполезен для агентного режима

### 2. Qwen2.5 Coder 7B
- Size: 4.7 GB
- Result: **agent mode формально включился**, Continue.dev предупреждает "Agent might not work well with this model"
- Verdict: слишком мала для агентного режима

### 3. Qwen2.5 Coder 14B
- Size: 9.0 GB
- Result: **agent mode сломан**
- Problem: вставляет сырой JSON tool call в открытый файл вместо создания нового
- Путь миграции: `migrations/001_...` вместо `infra/migrations/000001_...`
- Добавила trigger для updated_at — overengineering, игнорирует правила проекта
- Verdict: tool calling криво, системный промпт игнорируется

### 4. Qwen3 8B
- Size: 5.2 GB
- Verdict: не тестировали, по результатам других — 8B слишком мала

### 5. Qwen3 30B-A3B (MoE) — Continue.dev
- Size: 18 GB
- Architecture: Mixture-of-Experts, 30B параметров, 3B активных per prompt
- Result: **agent mode не работает в Continue.dev** — модель генерирует "размышления" (chain-of-thought) вместо вызова tools, не создаёт файлы
- Verdict: в Continue.dev agent mode не функционирует

### 6. Qwen3 30B-A3B (MoE) — Cline
- Size: 18 GB
- IDE: VS Code + Cline
- Result: **agent mode работает** — модель создала файлы, сгенерировала CRUD
- Проблемы с качеством:
  - Неправильная структура: `internal/models/` вместо `internal/model/`, `internal/repo/` вместо `internal/repository/`, `internal/api/` вместо `internal/handler/`
  - Перезаписала `cmd/main.go` вместо интеграции в существующий
  - Не использует sqlc — написала raw SQL через pgx напрямую (нарушает правила проекта)
  - Naming conventions нарушены (repo вместо repository, api вместо handler)
- Verdict: agent mode работает через Cline, но качество кода низкое — не следует системному промпту и конвенциям проекта

---

## IDE Tools Tested

### Continue.dev
- Тип: IDE плагин (VS Code / JetBrains)
- Agent mode: **не работает** с локальными моделями
- Проблемы:
  - Tool calling — модели не умеют корректно вызывать tools
  - Auto-approve — `experimental: autoApprove` не работает, только через UI
  - JSON injection — модели вставляют JSON tool calls в открытые файлы
  - System prompt — модели плохо следуют правилам

### Cline
- Тип: VS Code плагин
- Поддержка Ollama: да, без аккаунта
- Agent mode: **работает** с qwen3:30b-a3b — создаёт файлы, вызывает tools корректно
- Качество: низкое — не следует конвенциям проекта, неправильная структура папок
- Вывод: лучше Continue.dev для agent mode, но не заменяет облачный ассистент

### Ollama Agent (VS Code)
- Тип: community плагин (не официальный)
- Статус: рассмотрен, не установлен

### Нативный Ollama + VS Code Agent Mode
- Требует GitHub Copilot подписку
- Статус: не подходит

---

## Problems Encountered

### Continue.dev Agent Mode
1. **Tool calling** — модели не поддерживают или поддерживают криво
2. **Auto-approve** — настройка не работает через config, только UI
3. **JSON injection** — сырой JSON в открытый файл
4. **System prompt ignored** — модели не следуют правилам по путям и конвенциям
5. **Chain-of-thought leak** — qwen3:30b-a3b выводит внутренние размышления вместо tool calls

### Ollama / macOS
1. **Spotlight indexing** — скачивание моделей загружает CPU на 77%
2. **Auto-start** — отключается через System Settings → Login Items → Background Activity

---

## Models Considered But Not Installed

| Model | Size | Why considered | Why skipped |
|-------|------|----------------|-------------|
| codestral:22b | ~13 GB | Хорошая для кода | Нет tool calling |
| devstral:24b | ~15 GB | SWE-agent специализация | Не успели протестировать |
| deepseek-r1:32b | ~20 GB | Сильный reasoning | Тяжёлая, не код-специфичная |
| gpt-oss:20b | ~12 GB | Agent mode поддержка | Не успели протестировать |
| starcoder2:3b | ~2 GB | Автокомплит | Нужен только agent |
| qwen2.5-coder:1.5b | ~2 GB | Автокомплит | Нужен только agent |

---

## Final Config

### Ollama models
```
qwen3:30b-a3b    18 GB   — agent/chat
nomic-embed-text 274 MB  — embeddings
```

### ~/.continue/config.yaml
```yaml
name: Local Config
version: 1.0.0
schema: v1
models:
  - name: Qwen3 30B A3B
    provider: ollama
    model: qwen3:30b-a3b
    roles: [chat, edit, apply, autocomplete, summarize]
  - name: Embeddings
    provider: ollama
    model: nomic-embed-text
    roles: [embed]
context:
  - provider: code
  - provider: docs
  - provider: diff
  - provider: terminal
  - provider: problems
  - provider: folder
  - provider: codebase
```

### .continuerc.yaml (project system prompt)
```yaml
systemMessage: |
  BrandMoment — multi-tenant ad network. Monorepo: Go 1.23 + Next.js 15.
  Stack: chi router, pgx, sqlc, shadcn/ui, Tailwind v4, Postgres 17, BetterAuth (JWT).
  Go layers: handler → service → repository (wraps sqlc). DI via constructors.
  Multi-tenancy: sub-resources ALWAYS WHERE org_id = $1.
  Naming: snake_case files, PascalCase types, New* constructors.
  SQL: migrations up+down, IF NOT EXISTS, TIMESTAMPTZ.
```

---

## Sources
- [Continue.dev Agent Model Setup](https://docs.continue.dev/ide-extensions/agent/model-setup)
- [Qwen3 + Ollama + Continue.dev](https://medium.com/@rodrigo.estrada/build-a-local-ai-coding-assistant-qwen3-ollama-continue-dev-cee0dbcd172a)
- [Continue.dev + Ollama Setup](https://localaimaster.com/blog/continue-dev-ollama-setup)
- [Continue.dev + Codestral + Starcoder](https://abarbot.fr/ContinueOllama/continue_dev_ollama_codestral_starcoder_agent/)
- [Best Ollama Models 2026](https://www.morphllm.com/best-ollama-models)
- [Ollama Tool Calling](https://docs.ollama.com/capabilities/tool-calling)
- [Ollama Agent VS Code](https://marketplace.visualstudio.com/items?itemName=NishantUnavane.Ollama-Ai-agent)
- [Cline](https://marketplace.visualstudio.com/items?itemName=saoudrizwan.claude-dev)

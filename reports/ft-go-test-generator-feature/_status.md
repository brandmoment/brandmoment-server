# Task: Fine-Tuning Dataset for Go Test Generator

Profile: Feature  |  Stage: Implement (partial) — awaiting OpenAI key to run baseline
Created: 2026-04-20  |  Updated: 2026-04-20

## Context

Учебное задание по курсу LLM fine-tuning (из `task.txt`). Цель — собрать
датасет 50+ примеров и подготовить клиент для файнтюна по спеке OpenAI API.

Под задачу было решено тюнить генерацию **Go table-driven unit-тестов** в
стиле `services/api-dashboard/internal/repository/` для практической пользы
в проекте BrandMoment, а не синтетическую задачу «для галочки».

Изначально планировалось использовать Gemini AI Studio (бесплатно), но
Google задеприкейтил `tunedModels` endpoint — подтверждено raw REST-запросом
(501 UNIMPLEMENTED) и отсутствием `gemini-1.5-flash-001-tuning` в API (404).
Принято решение переехать на OpenAI напрямую (у пользователя есть доступ).

## Handoff

next: user  |  reason: нужен OPENAI_API_KEY для запуска baseline и (опц.) tune
input: `finetune/go-test-gen/` — весь код готов, dry-run верифицирован

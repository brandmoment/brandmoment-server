# Task: Confidence Estimation for LLM-based Rule Parser

Profile: Feature  |  Stage: Spec — ожидает утверждения развилок
Created: 2026-04-21  |  Updated: 2026-04-21

## Context

Учебное задание с курса LLM: «Оценка уверенности и контроль качества
инференса». Нужно реализовать механизм оценки уверенности LLM-ответа
**без файнтюна**, минимум 2 разных подхода из 4 (self-check, redundancy,
constraint-based, scoring).

Выбрана реальная задача проекта: **natural language → `PublisherRule`**
(парсинг фразы пользователя в структурированный JSON-конфиг правила
паблишера). Реализация на Go, интегрируется в `services/api-dashboard/`.

## Handoff

next: user  |  reason: нужно утвердить развилки из `01-spec.md` (SDK,
роутинг, API key) перед переходом в Implement
input: `reports/confidence-check-rule-parser/01-spec.md`

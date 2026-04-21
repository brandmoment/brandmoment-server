# Task: Fine-Tuning Dataset for Go Test Generator

Profile: Feature  |  Stage: Report — baseline done, awaiting video submission
Created: 2026-04-20  |  Updated: 2026-04-21

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

## Progress

- ✅ Датасет: 51 пара (метод → тест), train 41 / eval 10
- ✅ Валидатор: 0 ошибок
- ✅ Baseline (коммит `12cc671`): 10/10 ответов, средний балл 4.9/12
  (конвертеры ~10/12, ошибки БД ~2.7/12)
- ✅ Клиент файнтюна: готов, dry-run верифицирован (~$0.31 за 5 эпох)
- ⏳ Видео для сдачи курса
- ⏳ (Опц.) запуск реального тюна через `--execute`

## Handoff

next: user  |  reason: записать видео по чек-листу из `01-report.md` (шаг 6)
input: `finetune/go-test-gen/` + `reports/ft-go-test-generator-feature/01-report.md`

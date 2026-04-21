# Spec: Confidence Estimation для LLM Rule Parser

**Дата**: 2026-04-21
**Ветка**: `feature/confidence-check-rule-parser`
**Статус**: Спека, ожидает утверждения развилок

---

## Источник задачи

Курс по LLM, тема **«Оценка уверенности и контроль качества инференса»**.

Дословное требование:

> Возьмите задачу, где ошибка недопустима (классификация, извлечение
> данных, решение «да/нет», выбор действия). Реализуйте механизм оценки
> уверенности результата, **без fine-tuning**.
>
> Сделайте минимум 2 разных подхода из: Self-check, Redundancy,
> Constraint-based, Scoring.
>
> Протестируйте: корректные, пограничные, шумные входные данные.
> Замерьте: отклонения, повторные инференсы, latency, cost.

---

## Почему LLM вообще нужен в проекте (а не игрушка)

В `services/api-dashboard/` реализованы 5 типов `PublisherRule`:
`blocklist`, `allowlist`, `frequency_cap`, `geo_filter`, `platform_filter`.
Каждый имеет свой JSON-конфиг со строгой валидацией
(`service/publisher_rule.go:209-260`).

Чтобы создать правило, пользователь должен **руками собрать JSON**:

```json
{
  "type": "frequency_cap",
  "config": {"max_impressions": 3, "window_seconds": 86400}
}
```

UX — так себе. Идея фичи: **принимать фразу на естественном языке** и
парсить её в валидный `PublisherRule`:

> «Не показывай рекламу казино в РФ и Казахстане чаще 3 раз в сутки»

→ два правила: `geo_filter` (exclude RU, KZ) + `frequency_cap` (3 imp / 86400s).

Ошибка парсинга = реклама показана не туда, куда нужно = слитый бюджет
→ **классический «error-critical» сценарий**, подходит под требование
курса идеально.

---

## Варианты, которые рассматривали

| Вариант | Тип задачи | Почему не взяли |
|---|---|---|
| A — Модерация креативов (image/video safety) | Классификация | Нет реальной LLM-инфры в проекте, надо тащить vision-модель, нечего переиспользовать |
| B — Compliance-check текста рекламы | Да/нет + извлечение | Интереснее, но constraint-check слабый (нельзя формально проверить «текст не вводит в заблуждение») |
| **C — NL → PublisherRule** ✅ | Извлечение данных | Все 4 метода отлично ложатся, constraint-check **уже написан** (`validateRuleConfig`), реальная UX-фича |

---

## Почему Go, а не отдельный Python-скрипт

В прошлой задаче (fine-tuning) был Python — потому что задача офлайновая:
загрузить датасет, создать job, подождать. Код не шёл в продукт.

**Эта задача принципиально другая:**

1. **Инференс в рантайме** — вызов LLM каждый раз по запросу пользователя. Это **продовая логика**.
2. **Результат интегрируется в существующий код** — переиспользует `validateRuleConfig` (constraint-check уже написан).
3. **Нет long-running jobs** — только `chat.completions.create`. Реализуется одинаково просто на любом языке.
4. **Проект целиком на Go** — Python-огрызок сбоку создаст дыру в стеке.

**Плюсы Go-реализации:**
- Получаем настоящую фичу продукта, а не лабораторный скрипт
- Constraint-check уже готов (`service/publisher_rule.go:209`)
- Бенчмарк через `go test -bench` нативно
- Рецензент видит продакшн-интеграцию → больше очков

**Минусы (честно):**
- OpenAI Go SDK чуть менее полированный чем Python-ный. Но для
  `chat.completions.create` официального `github.com/openai/openai-go` хватит.

---

## Выбранные подходы к оценке уверенности

Берём **2 подхода** — минимум по заданию, максимум пользы.

### 1. Constraint-based
- Прогоняем JSON через `json.Unmarshal` → `validateRuleConfig`
- Enum для `type` (5 значений), обязательные поля, диапазоны (`max_impressions > 0`, `window_seconds > 0`), режимы (`include`/`exclude`)
- Ответ **мгновенно** — без дополнительных LLM-вызовов
- Результат: `OK` (прошёл валидацию) / `FAIL` (не прошёл)

### 2. Self-check
- Первый вызов: parse phrase → JSON
- Второй вызов: «посмотри на свой JSON и эту фразу, они соответствуют друг другу? ответь YES/NO + что не так»
- Если NO → `UNSURE`

---

## Предлагаемая структура

```
services/api-dashboard/
└── internal/
    ├── service/
    │   └── rule_parser.go          # Parse(phrase) → []PublisherRule + ConfidenceReport
    ├── handler/
    │   └── rule_parser.go          # POST /v1/publisher-rules/parse
    └── llm/
        ├── client.go               # ChatClient interface + provider enum
        ├── openai.go               # OpenAI реализация (openai-go)
        ├── gemini.go               # Gemini реализация (generative-ai-go)
        ├── self_check.go           # подход: self-check
        ├── redundancy.go           # подход: redundancy (3x parallel)
        ├── constraint.go           # подход: constraint (юзает validateRuleConfig)
        └── scoring.go              # подход: scoring (confidence 0.0–1.0)

apps/dashboard/
└── app/(dashboard)/
    └── publisher-rules/
        └── parse/
            └── page.tsx            # UI: textarea → parse → результат + confidence

finetune/confidence-check/          # отдельно: датасет + прогон метрик
├── data/
│   ├── corpus.jsonl                # 30+ фраз с эталонами
│   └── edge_cases.jsonl            # пограничные/шумные
├── benchmark_test.go               # Go-тест: прогоняет всё, пишет таблицу
├── results/
│   └── report.md                   # сравнение методов (latency, cost, reject rate)
└── README.md
```

---

## Датасет

Три группы фраз, **минимум 30 примеров суммарно**:

### Корректные (~15)
Одно правило на фразу, однозначная трактовка:
- «Блокируй категорию gambling» → `blocklist({categories: [gambling]})`
- «Разрешай только iOS» → `platform_filter(include, [ios])`
- «Показывай максимум 5 раз за час» → `frequency_cap(5, 3600)`

### Пограничные (~10)
Неоднозначная формулировка или составное правило:
- «Только премиум пользователи» — нет такого правила, ждём `UNSURE`
- «Не показывай казино в РФ чаще 3 раз в сутки» — **два правила сразу**
- «Разреши всё кроме gambling» — `blocklist`, но можно спутать с `allowlist`

### Шумные (~5)
Мусор, противоречия, опечатки:
- «kakashki» — просто мусор, ждём `FAIL`
- «Блокируй и разреши gambling одновременно» — противоречие
- «Показывай рекламу» — слишком общо, не тянет на правило

Формат эталона (`corpus.jsonl`):
```json
{
  "phrase": "Блокируй gambling в РФ",
  "expected": [
    {"type": "blocklist", "config": {"categories": ["gambling"]}},
    {"type": "geo_filter", "config": {"mode": "exclude", "country_codes": ["RU"]}}
  ],
  "expected_confidence": "OK",
  "group": "edge"
}
```

---

## Метрики (что замеряем)

| Метрика | Что означает |
|---|---|
| **Accuracy** | доля правильно распарсенных фраз (совпадение с эталоном) |
| **Reject rate** | сколько ответов помечено `FAIL` / `UNSURE` |
| **Retry rate** | сколько раз ушли на повторный инференс (redundancy делает х3 всегда) |
| **p50/p95 latency** | медиана и хвост по времени ответа |
| **Cost per call** | токены in/out × прайс `gpt-4o-mini` |
| **Calibration** | совпадение `confidence` с реальной правильностью |

Таблица сравнения по каждому из 4 подходов.

---

## Утверждённые решения (2026-04-21)

### 1. SDK → Оба провайдера за интерфейсом
- `github.com/openai/openai-go` (OpenAI)
- `github.com/google/generative-ai-go` (Gemini)
- Общий `ChatClient` интерфейс в `internal/llm/client.go`

### 2. Роутинг → A: подключаем сразу
- `POST /v1/publisher-rules/parse` в router
- Выбор провайдера через config / query param
- Если API-ключ не задан → 501

### 3. API-ключи → A: env
- `OPENAI_API_KEY` + `GEMINI_API_KEY` в `config.go`

### 4. Модели → Сравнительный бенчмарк
- `gpt-4o-mini` (OpenAI) + `gemini-2.0-flash` (Gemini)
- Бенчмарк прогоняет оба → сравнительная таблица

---

## Что делаем после утверждения

1. [Spec → Implement] Добавить `OpenAIAPIKey` в `config.go`, поднять `internal/llm/`
2. Написать `client.go` + `constraint.go` (быстро, переиспользуем готовое)
3. Написать `scoring.go`, `self_check.go`, `redundancy.go`
4. Собрать датасет в `finetune/confidence-check/data/`
5. Написать `benchmark_test.go` — прогон всех методов, таблица в `results/report.md`
6. [Test → Validate] `go test ./... ./finetune/...`
7. [Report] Финальный отчёт + видео для сдачи

---

## Оценка объёма

| Блок | Сложность | Время |
|---|---|---|
| Инфра `internal/llm/` + OpenAI client | Низкая | ~30 мин |
| 4 метода (constraint, scoring, self, redund) | Средняя | ~2 ч |
| Датасет 30+ фраз с эталонами | Средняя | ~1 ч |
| Benchmark harness + репорт | Средняя | ~1 ч |
| Полировка, юнит-тесты | Низкая | ~1 ч |
| **Итого** | | **~5–6 ч** |

Стоимость инференса в бенчмарке: ~$0.05–0.10 (30 фраз × 4 метода × средний промпт).

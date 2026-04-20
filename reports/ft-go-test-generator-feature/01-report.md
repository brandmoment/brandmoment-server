# Fine-Tuning Dataset для Go Test Generator — отчёт

**Дата**: 2026-04-20
**Ветка**: `feature/ft-go-test-generator`
**Коммиты**: `28acb00`, `80c10b7`
**Рабочая директория**: `finetune/go-test-gen/`

## Цель

Учебное задание — подготовить всю инфраструктуру для файнтюна LLM по спеке
из `task.txt`: датасет ≥50 примеров, валидатор, baseline, клиент для
запуска fine-tuning через API. Запускать сам тюн не требуется.

Под задачу выбрано тюнить **генерацию Go table-driven unit-тестов** в
стиле repository-слоя BrandMoment — для реальной пользы проекту, а не
синтетический toy-пример.

## Решения по задаче

| Вопрос | Выбор | Обоснование |
|---|---|---|
| Тип задачи | Генерация кода | `task.txt` вариант «код в нашем стеке» |
| Формат входа | Сигнатура + тело метода | Баланс точности и UX |
| Формат выхода | Только `func TestXxx(t *testing.T) {...}` | Стабильный короткий выход, без риска галлюцинации импортов |
| Слой тестов | `internal/repository/` | 9 файлов × ~6 тестов, чёткий паттерн, сильная проектная конвенция (org_id, mockDBTX) |
| Язык клиента | Python | `openai` SDK зрелый, минимум кода |
| Провайдер | OpenAI (`gpt-4o-mini`) | Буквальное соответствие `task.txt`, самый дешёвый API-тюн (~$0.25) |

## Что сделано

### Структура

```
finetune/go-test-gen/
├── data/
│   ├── raw/extracted.jsonl    # 51 пар (метод → тест)
│   ├── train.jsonl            # 41 пример (80%)
│   └── eval.jsonl             # 10 примеров (20%)
├── baseline/                  # пусто — ждём запуска
├── scripts/
│   ├── validate.py            # валидатор JSONL (роли, пустые content, JSON)
│   ├── extract_pairs.py       # парсер *.go + *_test.go → пары
│   ├── split.py               # стратифицированный 80/20 сплит
│   ├── baseline.py            # прогон eval через gpt-4o-mini без тюна
│   └── tune_client.py         # клиент: upload → job → poll (dry-run default)
├── criteria.md                # 12-балльная шкала оценки
├── README.md                  # инструкции
├── requirements.txt           # openai>=1.50.0
└── .gitignore
```

### Датасет

- **51 реальных пар** (метод, тест), извлечено автоматически из
  `services/api-dashboard/internal/repository/`
- **100% реальных данных** — требование task.txt ≥20% выполнено многократно
- **Сплит 41/10** (train/eval), стратифицированный по сущностям
- **Формат**: OpenAI `messages` (system + user + assistant) — ровно то, что
  ждёт OpenAI `/v1/files` для fine-tuning, без конверсий
- Валидатор: 0 ошибок

### Baseline

**Не запущен** — ждём `OPENAI_API_KEY`. Скрипт готов.

### Клиент файнтюна

Готов, dry-run проверен:
- 41 пример, ~20K токенов/эпоху, оценка стоимости **~$0.31 за 5 эпох**
- `--execute` флаг обязателен для реального запуска (защита от случайного)

## Путь, который не сработал (и почему)

Изначально пошли на **Gemini AI Studio** (бесплатно). Обнаружили:

1. `models/gemini-1.5-flash-001-tuning` → **HTTP 404** (модель удалена)
2. `/v1beta/tunedModels` → **HTTP 501 UNIMPLEMENTED** (endpoint выключен)
3. Baseline на free-tier: 5/10 — 503 UNAVAILABLE, 4/10 — 429 quota

Подтверждено через raw REST-запросы, новый SDK (`google-genai`) и legacy
SDK (`google-generativeai`, сам deprecated). Google перенёс тюн **только в
Vertex AI** (платный GCP).

Переписано под OpenAI — датасет не менялся (уже в нужном формате),
изменены только `baseline.py`, `tune_client.py`, `requirements.txt`, README.

## Безопасность

- В ходе работы пользователь случайно вставил Gemini API ключ в чат;
  ключ был помечен как скомпрометированный и ротирован до продолжения
- Все скрипты читают ключ **только из env** (`OPENAI_API_KEY`), без
  хардкода
- В локальном `.gitignore`: `.env`, `*.key`, `.venv/`

## Что осталось сделать

### 1. Получить OpenAI API-ключ и пополнить баланс

```bash
# 1. platform.openai.com → API keys → Create new secret key
# 2. Пополнить баланс: минимум $5 (хватит с запасом)
export OPENAI_API_KEY="sk-..."  # добавить в ~/.zprofile
```

### 2. Установить зависимости

```bash
python3 -m venv finetune/go-test-gen/.venv
finetune/go-test-gen/.venv/bin/pip install -r finetune/go-test-gen/requirements.txt
```

### 3. Запустить baseline (~30 секунд, ~$0.01)

```bash
finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/baseline.py \
    --eval finetune/go-test-gen/data/eval.jsonl \
    --out-md finetune/go-test-gen/baseline/responses.md \
    --out-jsonl finetune/go-test-gen/baseline/responses.jsonl
```

### 4. Заполнить чек-листы в `baseline/responses.md`

По каждому из 10 ответов отметить пункты из `criteria.md` (компилируется ли,
правильный ли нейминг, использует ли `mockDBTX` и т.п.).

### 5. (Опционально для сдачи) Прогнать реальный тюн

Формально `task.txt` **запрещает** запускать («пока не запускайте — только
подготовьте код»). Но если хочется для своего эксперимента:

```bash
finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/tune_client.py \
    --train finetune/go-test-gen/data/train.jsonl --execute
```

Займёт 10–30 минут, стоит ~$0.25, результат — `ft:gpt-4o-mini:...:go-test-gen:...`.

### 6. Записать видео для сдачи

Покажи:
1. Структуру `finetune/go-test-gen/`
2. `validate.py` на `train.jsonl` (OK, 41 valid)
3. Экстрактор `extract_pairs.py` — как собирал датасет
4. Пример из `train.jsonl` (один JSON-объект раскрытый)
5. `baseline/responses.md` — сравнение с эталоном, чек-листы
6. `criteria.md` — критерии оценки
7. `tune_client.py` в dry-run режиме — показать что план готов
8. (Опц.) `--execute` — но запустить уже после съёмки

### 7. Сдать

Ссылка на ветку `feature/ft-go-test-generator` + видео.

## Метрики (для отчёта после сдачи)

| | План | Факт |
|---|---|---|
| Примеров в датасете | ≥50 | **51** |
| Доля реальных | ≥20% | **100%** |
| Train/eval split | 80/20 | **41/10** |
| Baseline прогон | 10 | ждёт ключа |
| Клиент файнтюна | готов, не запущен | **✅ готов** |
| Стоимость задачи | n/a | <$0.50 |

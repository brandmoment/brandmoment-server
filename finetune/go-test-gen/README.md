# Go Test Generator — Fine-Tuning Dataset

Цель — дообучить `gpt-4o-mini` генерировать table-driven unit-тесты в стиле repository-слоя BrandMoment.

## Содержание

```
finetune/go-test-gen/
├── data/
│   ├── raw/extracted.jsonl    # 51 пара (метод, тест) из internal/repository
│   ├── train.jsonl            # 41 пример (80%)
│   └── eval.jsonl             # 10 примеров (20%)
├── baseline/
│   ├── responses.md           # ответы БАЗОВОЙ модели без тюна (для сравнения)
│   └── responses.jsonl        # то же в JSON
├── scripts/
│   ├── validate.py            # проверка JSONL (роли, пустые content, валидный JSON)
│   ├── extract_pairs.py       # парсит *_test.go + *.go и собирает пары
│   ├── split.py               # стратифицированный 80/20 сплит
│   ├── baseline.py            # прогоняет eval через базовую модель
│   └── tune_client.py         # клиент файнтюна (upload → job → poll)
├── criteria.md                # критерии «стало лучше»
└── README.md
```

## Задача (из task.txt)

- Тип: **генерация** (код в нашем стеке)
- Датасет: ≥ 50 примеров, ≥ 20% реальных — у нас **100% реальных** (51 пара).
- Формат: OpenAI `messages` (system + user + assistant), как того требует спек.
- Baseline: 10 eval через `gpt-4o-mini` без тюна.
- Клиент: подготовлен, **не запускается без явного `--execute` флага**.

## Модели

| Назначение | Модель |
|---|---|
| Baseline (без тюна) | `gpt-4o-mini` |
| База для тюна | `gpt-4o-mini-2024-07-18` |

## Стоимость

- Baseline (10 inference запросов) — ~$0.01
- Fine-tuning одного прогона (41 пример × 5 эпох) — ~$0.25
- Инференс тюненной модели на 10 eval — ~$0.01

**Итого на всю задачу — меньше $0.50.**

## Воспроизведение

```bash
# 1. Создать venv и поставить SDK
python3 -m venv finetune/go-test-gen/.venv
finetune/go-test-gen/.venv/bin/pip install -r finetune/go-test-gen/requirements.txt

# 2. Положить ключ в env
export OPENAI_API_KEY="sk-..."

# 3. (опционально) Пересобрать датасет из свежих тестов
finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/extract_pairs.py \
    --source services/api-dashboard/internal/repository \
    --out finetune/go-test-gen/data/raw/extracted.jsonl

finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/split.py \
    --input finetune/go-test-gen/data/raw/extracted.jsonl \
    --train finetune/go-test-gen/data/train.jsonl \
    --eval  finetune/go-test-gen/data/eval.jsonl \
    --eval-size 10 --seed 42

# 4. Провалидировать
finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/validate.py \
    finetune/go-test-gen/data/train.jsonl \
    finetune/go-test-gen/data/eval.jsonl

# 5. Замерить baseline (~30 секунд)
finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/baseline.py \
    --eval finetune/go-test-gen/data/eval.jsonl \
    --out-md finetune/go-test-gen/baseline/responses.md \
    --out-jsonl finetune/go-test-gen/baseline/responses.jsonl

# 6. Посмотреть план тюна (ничего не отправляется)
finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/tune_client.py \
    --train finetune/go-test-gen/data/train.jsonl

# 7. Запустить тюн по-настоящему (для сдачи — задание запрещает, делай только если хочешь проверить)
# finetune/go-test-gen/.venv/bin/python finetune/go-test-gen/scripts/tune_client.py \
#     --train finetune/go-test-gen/data/train.jsonl --execute
```

## Формат данных

Каждая строка в JSONL:

```json
{"messages": [
    {"role": "system", "content": "You are a Go test generator for BrandMoment..."},
    {"role": "user", "content": "func (r *apiKeyRepo) GetByID(...) (*model.APIKey, error) { ... }\n\n// Scenario: NotFound"},
    {"role": "assistant", "content": "func TestAPIKeyRepo_GetByID_NotFound(t *testing.T) { ... }"}
]}
```

Формат **ровно такой**, какой требует OpenAI Fine-tuning API — загружается в `/v1/files` как есть, без конверсий.

## Критерии оценки

См. [criteria.md](criteria.md). Коротко: 12-балльная шкала по трём блокам (формальная корректность, структурное сходство с эталоном, project-specific конвенции). Цель файнтюна — `tuned_avg ≥ baseline_avg + 4`.

## Безопасность

- API-ключ — только из `$OPENAI_API_KEY` env var; в код не попадает.
- `.env`, `*.key`, `.venv` — в локальном `.gitignore`.

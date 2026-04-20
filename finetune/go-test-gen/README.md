# Go Test Generator — Fine-Tuning Dataset

Цель — обучить `gemini-1.5-flash-001` генерировать table-driven unit-тесты в стиле repository-слоя BrandMoment.

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
- Формат: OpenAI messages (system + user + assistant), как того требует спек.
- Baseline: 10 eval через `gemini-1.5-flash-001` без тюна.
- Клиент: подготовлен, **не запускался**.

## Модели

| Назначение | Модель |
|---|---|
| Baseline (без тюна) | `models/gemini-1.5-flash-001` |
| База для тюна | `models/gemini-1.5-flash-001-tuning` |

**Почему Gemini, а не gpt-4o-mini**: нет доступа к OpenAI API. Датасет хранится в OpenAI-совместимом формате (task.txt спек), `tune_client.py` конвертирует в Gemini-формат `{text_input, output}` на лету.

## Воспроизведение

```bash
# 1. Установить зависимости
pip install google-genai

# 2. Положить ключ в env (ключ получить на aistudio.google.com)
export GEMINI_API_KEY="AIza..."

# 3. (опционально) Пересобрать датасет из свежих тестов
python scripts/extract_pairs.py \
    --source ../../services/api-dashboard/internal/repository \
    --out data/raw/extracted.jsonl

python scripts/split.py \
    --input data/raw/extracted.jsonl \
    --train data/train.jsonl \
    --eval  data/eval.jsonl \
    --eval-size 10 --seed 42

# 4. Провалидировать
python scripts/validate.py data/train.jsonl data/eval.jsonl

# 5. Замерить baseline (~1 минута, 4.5s между запросами ради RPM-лимитов)
python scripts/baseline.py \
    --eval data/eval.jsonl \
    --out-md baseline/responses.md \
    --out-jsonl baseline/responses.jsonl

# 6. Посмотреть план тюна (ничего не отправляется)
python scripts/tune_client.py --train data/train.jsonl

# 7. Запустить тюн по-настоящему (НЕ для сдачи — только если хочешь проверить)
# python scripts/tune_client.py --train data/train.jsonl --execute
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

## Критерии оценки

См. [criteria.md](criteria.md). Коротко: 12-балльная шкала по трём блокам (формальная корректность, структурное сходство с эталоном, project-specific конвенции). Цель файнтюна — `tuned_avg ≥ baseline_avg + 4`.

## Безопасность

- API-ключ — только из `$GEMINI_API_KEY` env var; в код не попадает.
- `.env` и `*.key` — в локальном `.gitignore`.
- Исходный ключ, засветившийся в истории переписки, **был ротирован** до запуска.

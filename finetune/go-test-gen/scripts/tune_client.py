"""
Fine-tuning client for OpenAI.

Automates the three required steps from task.txt:
    1. upload file  — upload train.jsonl with purpose="fine-tune"
    2. create job   — start a fine-tuning job on gpt-4o-mini
    3. poll status  — wait until the job reaches a terminal state

By default the script runs in --dry-run mode: it validates the dataset,
prints what would be uploaded, but does NOT contact the API. Use --execute
to actually submit (costs ~$0.25 for our 41-example dataset; job takes
10–30 minutes to complete).

Usage:
    export OPENAI_API_KEY=sk-...
    pip install -r requirements.txt

    # Plan-only (safe):
    python scripts/tune_client.py --train data/train.jsonl

    # Real run:
    python scripts/tune_client.py --train data/train.jsonl --execute
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from pathlib import Path

DEFAULT_BASE_MODEL = "gpt-4o-mini-2024-07-18"
DEFAULT_SUFFIX = "go-test-gen"
POLL_INTERVAL_SEC = 30
TERMINAL_STATES = {"succeeded", "failed", "cancelled"}


def load_jsonl(path: Path) -> list[dict]:
    with path.open("r", encoding="utf-8") as fh:
        return [json.loads(line) for line in fh if line.strip()]


def print_plan(base_model: str, suffix: str, train_path: Path, examples: list[dict]) -> None:
    total_chars = sum(
        sum(len(m["content"]) for m in ex["messages"]) for ex in examples
    )
    approx_tokens = total_chars // 4  # ~4 chars per token, rough estimate
    print("=" * 60)
    print("FINE-TUNING PLAN (dry-run — nothing sent to API)")
    print("=" * 60)
    print(f"base model:       {base_model}")
    print(f"name suffix:      {suffix}")
    print(f"training file:    {train_path}")
    print(f"examples:         {len(examples)}")
    print(f"approx. tokens:   ~{approx_tokens:,} (for a single epoch)")
    print(f"estimated cost:   ~${approx_tokens * 5 * 3 / 1_000_000:.2f} for 5 epochs @ $3/1M")
    if examples:
        first = examples[0]["messages"]
        roles = [m["role"] for m in first]
        print(f"first example roles: {roles}")
    print("\nto execute for real, re-run with --execute")
    print("=" * 60)


def run(
    train_path: Path,
    base_model: str,
    suffix: str,
) -> int:
    api_key = os.environ.get("OPENAI_API_KEY")
    if not api_key:
        print("error: OPENAI_API_KEY is not set", file=sys.stderr)
        return 2

    try:
        from openai import OpenAI
    except ImportError:
        print("error: openai is not installed. Run: pip install -r requirements.txt", file=sys.stderr)
        return 2

    client = OpenAI(api_key=api_key)

    print(f"[1/3] uploading {train_path} (purpose=fine-tune)...")
    with train_path.open("rb") as fh:
        uploaded = client.files.create(file=fh, purpose="fine-tune")
    print(f"      file id: {uploaded.id}")

    print(f"[2/3] creating fine-tuning job (base={base_model}, suffix={suffix})...")
    job = client.fine_tuning.jobs.create(
        training_file=uploaded.id,
        model=base_model,
        suffix=suffix,
    )
    print(f"      job id: {job.id}")
    print(f"      initial status: {job.status}")

    print(f"[3/3] polling status every {POLL_INTERVAL_SEC}s...")
    while job.status not in TERMINAL_STATES:
        time.sleep(POLL_INTERVAL_SEC)
        job = client.fine_tuning.jobs.retrieve(job.id)
        print(f"      status: {job.status}  trained_tokens={getattr(job, 'trained_tokens', None)}")

    print(f"done. terminal status: {job.status}")
    if job.fine_tuned_model:
        print(f"      tuned model id: {job.fine_tuned_model}")
        print("      use this id in chat.completions.create(model=...) at inference time.")
    else:
        print("      no tuned model produced — inspect the job for errors:")
        if getattr(job, "error", None):
            print(f"      {job.error}")

    return 0 if job.status == "succeeded" else 1


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--train", type=Path, required=True)
    parser.add_argument("--base-model", default=DEFAULT_BASE_MODEL)
    parser.add_argument("--suffix", default=DEFAULT_SUFFIX, help="appears in the fine-tuned model id")
    parser.add_argument(
        "--execute",
        action="store_true",
        help="actually submit to the API (default is dry-run / plan only)",
    )
    args = parser.parse_args()

    examples = load_jsonl(args.train)

    if not args.execute:
        print_plan(args.base_model, args.suffix, args.train, examples)
        return 0

    return run(
        train_path=args.train,
        base_model=args.base_model,
        suffix=args.suffix,
    )


if __name__ == "__main__":
    sys.exit(main())

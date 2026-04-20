"""
Fine-tuning client for Gemini.

Automates the three required steps from task.txt:
    1. upload file   — converts our OpenAI-messages JSONL to Gemini's
                       text_input/output format and uploads the dataset
    2. create job    — starts a tuning job on `gemini-1.5-flash-001-tuning`
    3. poll status   — waits until the job reaches a terminal state

By default this script runs in --dry-run mode: it loads the data, prints
what would be uploaded, but does NOT contact the API. Use --execute to
actually submit the job (costs quota on the free tier; an active tuning
job can take 20–60 minutes to finish).

Usage:
    export GEMINI_API_KEY=...
    pip install google-genai

    # Plan-only (safe):
    python scripts/tune_client.py --train data/train.jsonl

    # Real run (submits job, pools every 30s):
    python scripts/tune_client.py --train data/train.jsonl --execute
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from pathlib import Path

DEFAULT_BASE_MODEL = "models/gemini-1.5-flash-001-tuning"
DEFAULT_DISPLAY_NAME = "go-test-gen-v1"
POLL_INTERVAL_SEC = 30
TERMINAL_STATES = {
    "JOB_STATE_SUCCEEDED",
    "JOB_STATE_FAILED",
    "JOB_STATE_CANCELLED",
    "JOB_STATE_EXPIRED",
}


def load_openai_jsonl(path: Path) -> list[dict]:
    """Load our OpenAI-messages-style JSONL."""
    out: list[dict] = []
    with path.open("r", encoding="utf-8") as fh:
        for line in fh:
            if line.strip():
                out.append(json.loads(line))
    return out


def to_gemini_examples(openai_examples: list[dict]) -> list[dict]:
    """Convert {messages:[system,user,assistant]} into Gemini tuning pairs.

    Gemini tuning does NOT train on system instructions. We merge the system
    prompt into the user input so the tuned model learns the same mapping;
    at inference time the caller should pass the same system_instruction via
    GenerateContentConfig.
    """
    result: list[dict] = []
    for ex in openai_examples:
        msgs = {m["role"]: m["content"] for m in ex["messages"]}
        user = msgs.get("user", "")
        assistant = msgs.get("assistant", "")
        result.append({"text_input": user, "output": assistant})
    return result


def print_plan(base_model: str, display_name: str, examples: list[dict]) -> None:
    print("=" * 60)
    print("FINE-TUNING PLAN (dry-run — nothing sent to API)")
    print("=" * 60)
    print(f"base model:    {base_model}")
    print(f"display name:  {display_name}")
    print(f"examples:      {len(examples)}")
    if examples:
        preview = examples[0]
        print("\nfirst example preview:")
        print(f"  text_input ({len(preview['text_input'])} chars):")
        print("    " + preview["text_input"].splitlines()[0][:100])
        print(f"  output ({len(preview['output'])} chars):")
        print("    " + preview["output"].splitlines()[0][:100])
    print("\nto execute for real, re-run with --execute")
    print("=" * 60)


def run(
    train_path: Path,
    base_model: str,
    display_name: str,
    epoch_count: int,
    batch_size: int,
    learning_rate: float,
) -> int:
    api_key = os.environ.get("GEMINI_API_KEY")
    if not api_key:
        print("error: GEMINI_API_KEY is not set", file=sys.stderr)
        return 2

    try:
        from google import genai
        from google.genai import types
    except ImportError:
        print("error: google-genai is not installed. Run: pip install google-genai", file=sys.stderr)
        return 2

    openai_examples = load_openai_jsonl(train_path)
    gemini_examples = to_gemini_examples(openai_examples)

    client = genai.Client(api_key=api_key)

    print(f"[1/3] submitting tuning job for {len(gemini_examples)} example(s)...")
    dataset = types.TuningDataset(
        examples=[types.TuningExample(**ex) for ex in gemini_examples],
    )
    job = client.tunings.tune(
        base_model=base_model,
        training_dataset=dataset,
        config=types.CreateTuningJobConfig(
            tuned_model_display_name=display_name,
            epoch_count=epoch_count,
            batch_size=batch_size,
            learning_rate=learning_rate,
        ),
    )
    print(f"      job created: name={job.name}")
    print(f"      initial state: {job.state}")

    print("[2/3] polling status every {}s...".format(POLL_INTERVAL_SEC))
    while str(job.state) not in TERMINAL_STATES and getattr(job.state, "name", "") not in TERMINAL_STATES:
        time.sleep(POLL_INTERVAL_SEC)
        job = client.tunings.get(name=job.name)
        print(f"      state: {job.state}")

    print(f"[3/3] terminal state reached: {job.state}")
    tuned = getattr(job, "tuned_model", None)
    if tuned is not None and getattr(tuned, "model", None):
        print(f"      tuned model id: {tuned.model}")
        print(f"      use this id at inference time in generate_content(model=...)")
    else:
        print("      no tuned model produced — inspect the job for errors")

    return 0 if getattr(job.state, "name", str(job.state)).endswith("SUCCEEDED") else 1


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--train", type=Path, required=True)
    parser.add_argument("--base-model", default=DEFAULT_BASE_MODEL)
    parser.add_argument("--display-name", default=DEFAULT_DISPLAY_NAME)
    parser.add_argument("--epoch-count", type=int, default=5)
    parser.add_argument("--batch-size", type=int, default=4)
    parser.add_argument("--learning-rate", type=float, default=0.001)
    parser.add_argument(
        "--execute",
        action="store_true",
        help="actually submit to the API (default is dry-run / plan only)",
    )
    args = parser.parse_args()

    openai_examples = load_openai_jsonl(args.train)
    gemini_examples = to_gemini_examples(openai_examples)

    if not args.execute:
        print_plan(args.base_model, args.display_name, gemini_examples)
        return 0

    return run(
        train_path=args.train,
        base_model=args.base_model,
        display_name=args.display_name,
        epoch_count=args.epoch_count,
        batch_size=args.batch_size,
        learning_rate=args.learning_rate,
    )


if __name__ == "__main__":
    sys.exit(main())

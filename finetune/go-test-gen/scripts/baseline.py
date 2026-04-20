"""
Baseline runner: feed every eval example to the BASE Gemini model (no tuning)
and save its responses. This produces the "before" snapshot we will compare
against the fine-tuned model later.

Produces:
    baseline/responses.md   — human-readable side-by-side (input / reference / baseline)
    baseline/responses.jsonl — machine-readable copy for later scoring

Usage:
    export GEMINI_API_KEY=...
    pip install google-genai
    python scripts/baseline.py \
        --eval data/eval.jsonl \
        --out-md baseline/responses.md \
        --out-jsonl baseline/responses.jsonl \
        --model gemini-1.5-flash-001
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--eval", type=Path, required=True)
    parser.add_argument("--out-md", type=Path, required=True)
    parser.add_argument("--out-jsonl", type=Path, required=True)
    parser.add_argument(
        "--model",
        default="gemini-1.5-flash-001",
        help="base model id (must match the tuning base, minus the -tuning suffix)",
    )
    parser.add_argument(
        "--sleep",
        type=float,
        default=4.5,
        help="seconds between requests to stay under free-tier RPM limits",
    )
    args = parser.parse_args()

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

    client = genai.Client(api_key=api_key)

    examples = []
    with args.eval.open("r", encoding="utf-8") as fh:
        for line in fh:
            if line.strip():
                examples.append(json.loads(line))

    print(f"running baseline on {len(examples)} example(s) with model={args.model}")

    results = []
    for idx, ex in enumerate(examples, start=1):
        system = ex["messages"][0]["content"]
        user = ex["messages"][1]["content"]
        reference = ex["messages"][2]["content"]

        print(f"  [{idx}/{len(examples)}] calling model...", flush=True)
        try:
            resp = client.models.generate_content(
                model=f"models/{args.model}",
                contents=user,
                config=types.GenerateContentConfig(
                    system_instruction=system,
                    temperature=0.2,
                ),
            )
            output = (resp.text or "").strip()
            error = None
        except Exception as exc:  # noqa: BLE001 — we want any failure captured
            output = ""
            error = str(exc)

        results.append({
            "index": idx,
            "user": user,
            "reference": reference,
            "baseline_output": output,
            "error": error,
        })

        if idx < len(examples):
            time.sleep(args.sleep)

    args.out_md.parent.mkdir(parents=True, exist_ok=True)
    args.out_jsonl.parent.mkdir(parents=True, exist_ok=True)

    with args.out_jsonl.open("w", encoding="utf-8") as fh:
        for r in results:
            fh.write(json.dumps(r, ensure_ascii=False) + "\n")

    with args.out_md.open("w", encoding="utf-8") as fh:
        fh.write(f"# Baseline — {args.model} (no tuning)\n\n")
        fh.write(f"Source eval: `{args.eval}`\n\n")
        fh.write(f"Run: {len(results)} example(s)\n\n---\n\n")
        for r in results:
            fh.write(f"## Example {r['index']}\n\n")
            fh.write("### Input (user)\n\n```go\n")
            fh.write(r["user"].strip() + "\n```\n\n")
            fh.write("### Reference (expected)\n\n```go\n")
            fh.write(r["reference"].strip() + "\n```\n\n")
            fh.write("### Baseline output\n\n")
            if r["error"]:
                fh.write(f"**ERROR:** {r['error']}\n\n")
            else:
                fh.write("```go\n" + r["baseline_output"] + "\n```\n\n")
            fh.write("### Notes (fill in manually)\n\n")
            fh.write("- Compiles: [ ]\n")
            fh.write("- Uses mock_dbtx helpers correctly: [ ]\n")
            fh.write("- Naming matches TestType_Method[_Scenario]: [ ]\n")
            fh.write("- org_id / scope params covered: [ ]\n")
            fh.write("- Comments:\n\n---\n\n")

    ok = sum(1 for r in results if not r["error"])
    print(f"done. {ok}/{len(results)} succeeded. wrote {args.out_md} and {args.out_jsonl}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

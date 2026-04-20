"""
Split an extracted JSONL into train/eval files.

Deterministic with a seed so re-runs keep the same split. Uses a stratified-ish
strategy: one example per source file is reserved for eval first, then the
remaining eval slots are filled at random. This guarantees eval covers multiple
entity types.

Usage:
    python scripts/split.py \
        --input data/raw/extracted.jsonl \
        --train data/train.jsonl \
        --eval  data/eval.jsonl \
        --eval-size 10 \
        --seed 42
"""

from __future__ import annotations

import argparse
import json
import random
import re
import sys
from collections import defaultdict
from pathlib import Path


# Extract the "entity" token from the test function name so we can stratify
# by entity: TestAPIKeyRepo_X -> apikey, TestOrgInviteRepo_X -> orginvite, etc.
TEST_NAME_RE = re.compile(r"^func\s+(Test\w+)")


def entity_key(assistant_text: str) -> str:
    m = TEST_NAME_RE.search(assistant_text)
    if not m:
        return "_unknown"
    name = m.group(1)
    tail = name[len("Test"):]
    head = tail.split("_", 1)[0]
    return head.lower().replace("repo", "")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--train", type=Path, required=True)
    parser.add_argument("--eval", type=Path, required=True)
    parser.add_argument("--eval-size", type=int, default=10)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    lines = [line for line in args.input.read_text(encoding="utf-8").splitlines() if line.strip()]
    if len(lines) < args.eval_size + 1:
        print(
            f"error: not enough examples ({len(lines)}) for eval-size={args.eval_size}",
            file=sys.stderr,
        )
        return 2

    buckets: dict[str, list[int]] = defaultdict(list)
    for idx, raw in enumerate(lines):
        ex = json.loads(raw)
        key = entity_key(ex["messages"][2]["content"])
        buckets[key].append(idx)

    rng = random.Random(args.seed)

    # Reserve one example from each bucket for eval, shuffled.
    eval_idx: list[int] = []
    for key in sorted(buckets.keys()):
        pool = buckets[key][:]
        rng.shuffle(pool)
        eval_idx.append(pool[0])
        if len(eval_idx) >= args.eval_size:
            break

    # Fill the rest of eval from whatever remains.
    remaining = [i for i in range(len(lines)) if i not in eval_idx]
    rng.shuffle(remaining)
    while len(eval_idx) < args.eval_size and remaining:
        eval_idx.append(remaining.pop())

    eval_set = set(eval_idx)
    train_idx = [i for i in range(len(lines)) if i not in eval_set]

    args.train.parent.mkdir(parents=True, exist_ok=True)
    args.eval.parent.mkdir(parents=True, exist_ok=True)

    args.train.write_text(
        "\n".join(lines[i] for i in train_idx) + "\n",
        encoding="utf-8",
    )
    args.eval.write_text(
        "\n".join(lines[i] for i in eval_idx) + "\n",
        encoding="utf-8",
    )

    print(f"train: {len(train_idx)} examples -> {args.train}")
    print(f"eval:  {len(eval_idx)} examples -> {args.eval}")

    # Report eval entity distribution so we can eyeball balance.
    eval_buckets: dict[str, int] = defaultdict(int)
    for i in eval_idx:
        ex = json.loads(lines[i])
        eval_buckets[entity_key(ex["messages"][2]["content"])] += 1
    print("eval by entity:", dict(sorted(eval_buckets.items())))
    return 0


if __name__ == "__main__":
    sys.exit(main())

"""
Dataset validator for the fine-tuning task.

Validates a JSONL file against the spec from task.txt:
    - every line is a valid JSON object;
    - each object has a "messages" array with exactly three entries;
    - roles appear in order: system, user, assistant;
    - no "content" field is empty or whitespace-only.

Usage:
    python scripts/validate.py data/train.jsonl
    python scripts/validate.py data/train.jsonl data/eval.jsonl

Exit code 0 if every file is valid, 1 otherwise.
"""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass
from pathlib import Path

EXPECTED_ROLES = ("system", "user", "assistant")


@dataclass
class Issue:
    line: int
    message: str


def validate_line(raw: str, line_no: int) -> list[Issue]:
    issues: list[Issue] = []

    try:
        obj = json.loads(raw)
    except json.JSONDecodeError as exc:
        return [Issue(line_no, f"invalid JSON: {exc.msg} at col {exc.colno}")]

    if not isinstance(obj, dict):
        return [Issue(line_no, f"top-level must be an object, got {type(obj).__name__}")]

    messages = obj.get("messages")
    if not isinstance(messages, list):
        return [Issue(line_no, "missing or non-list 'messages' field")]

    if len(messages) != len(EXPECTED_ROLES):
        issues.append(Issue(
            line_no,
            f"expected {len(EXPECTED_ROLES)} messages, got {len(messages)}",
        ))

    for idx, (msg, expected_role) in enumerate(zip(messages, EXPECTED_ROLES)):
        prefix = f"messages[{idx}]"
        if not isinstance(msg, dict):
            issues.append(Issue(line_no, f"{prefix}: must be an object"))
            continue

        role = msg.get("role")
        if role != expected_role:
            issues.append(Issue(
                line_no,
                f"{prefix}.role: expected '{expected_role}', got {role!r}",
            ))

        content = msg.get("content")
        if not isinstance(content, str):
            issues.append(Issue(line_no, f"{prefix}.content: must be a string"))
        elif not content.strip():
            issues.append(Issue(line_no, f"{prefix}.content: must not be empty"))

    return issues


def validate_file(path: Path) -> list[Issue]:
    if not path.exists():
        return [Issue(0, f"file not found: {path}")]

    issues: list[Issue] = []
    with path.open("r", encoding="utf-8") as fh:
        for line_no, raw in enumerate(fh, start=1):
            stripped = raw.strip()
            if not stripped:
                issues.append(Issue(line_no, "empty line (JSONL requires one object per line)"))
                continue
            issues.extend(validate_line(stripped, line_no))

    return issues


def count_lines(path: Path) -> int:
    with path.open("r", encoding="utf-8") as fh:
        return sum(1 for line in fh if line.strip())


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print("usage: python validate.py <file.jsonl> [<file.jsonl> ...]", file=sys.stderr)
        return 2

    overall_ok = True
    for raw_path in argv[1:]:
        path = Path(raw_path)
        issues = validate_file(path)
        total = count_lines(path) if path.exists() else 0

        if issues:
            overall_ok = False
            print(f"[FAIL] {path} — {len(issues)} issue(s):")
            for issue in issues:
                print(f"  line {issue.line}: {issue.message}")
        else:
            print(f"[OK]   {path} — {total} valid example(s)")

    return 0 if overall_ok else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))

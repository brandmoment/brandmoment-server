"""
Extract (method, test) pairs from internal/repository for dataset seeding.

Scans every `internal/repository/*_test.go` in the given root, pairs each
`func TestXxx(t *testing.T)` with the source function it tests, and writes
JSONL examples ready for fine-tuning (OpenAI chat-messages format).

Matching rules:
    TestToCamelCase                           -> func toCamelCase(...)
    TestReceiverType_Method[_Scenario]        -> func (r *receiverType) Method(...)

When a test's name carries a Scenario suffix (NotFound, DBError, ...),
it is included in the user prompt as a hint, so the model can learn to
produce scenario-specific tests.

Usage:
    python scripts/extract_pairs.py \
        --source services/api-dashboard/internal/repository \
        --out data/raw/extracted.jsonl
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path

SYSTEM_PROMPT = (
    "You are a Go test generator for the BrandMoment project (multi-tenant ad network). "
    "Given a Go function (typically a repository method or a type converter) and an "
    "optional scenario hint, produce a single unit test function in the project's "
    "style: named TestTypeName_Method[_Scenario] for methods or TestFuncName for "
    "free functions; table-driven with tests := []struct{...}{} when multiple cases "
    "apply; use mock_dbtx_test.go helpers (mockDBTX, errRow, errSentinel) for DB "
    "error paths; assert with t.Errorf / t.Fatalf; no package/import lines — just "
    "the function body."
)


@dataclass
class GoFunc:
    """A parsed Go top-level function declaration."""
    name: str
    receiver_type: str | None  # e.g. "apiKeyRepo" for pointer receiver, or None for free func
    signature: str             # full declaration line up to the opening brace (inclusive)
    body: str                  # body between the outermost braces, inclusive
    full: str                  # signature + body (ready to use as code block)


# Matches "func Name(" or "func (r *Type) Name(" or "func (r Type) Name(" at start of line.
FUNC_HEADER_RE = re.compile(
    r"^func\s+(?:\((?P<recv>[^)]+)\)\s+)?(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*\(",
    re.MULTILINE,
)


def parse_functions(src: str) -> list[GoFunc]:
    """Parse every top-level function in a Go source file."""
    funcs: list[GoFunc] = []
    for match in FUNC_HEADER_RE.finditer(src):
        start = match.start()

        # Find the opening brace of the function body.
        brace_idx = src.find("{", match.end() - 1)
        if brace_idx == -1:
            continue

        # Walk the braces to locate the matching close. String/rune/comment
        # handling is enough to survive our codebase.
        depth = 0
        i = brace_idx
        in_string = False       # inside "..."
        in_rune = False         # inside '.'
        in_line_comment = False
        in_block_comment = False
        end_idx = -1
        while i < len(src):
            ch = src[i]
            nxt = src[i + 1] if i + 1 < len(src) else ""

            if in_line_comment:
                if ch == "\n":
                    in_line_comment = False
            elif in_block_comment:
                if ch == "*" and nxt == "/":
                    in_block_comment = False
                    i += 1
            elif in_string:
                if ch == "\\":
                    i += 1  # skip escaped char
                elif ch == '"':
                    in_string = False
            elif in_rune:
                if ch == "\\":
                    i += 1
                elif ch == "'":
                    in_rune = False
            else:
                if ch == "/" and nxt == "/":
                    in_line_comment = True
                    i += 1
                elif ch == "/" and nxt == "*":
                    in_block_comment = True
                    i += 1
                elif ch == '"':
                    in_string = True
                elif ch == "'":
                    in_rune = True
                elif ch == "`":
                    # Raw string — skip to next backtick.
                    close = src.find("`", i + 1)
                    i = close if close != -1 else len(src)
                elif ch == "{":
                    depth += 1
                elif ch == "}":
                    depth -= 1
                    if depth == 0:
                        end_idx = i
                        break
            i += 1

        if end_idx == -1:
            continue

        signature = src[start:brace_idx + 1]
        body = src[brace_idx:end_idx + 1]
        full = src[start:end_idx + 1]

        receiver_type: str | None = None
        recv_raw = match.group("recv")
        if recv_raw:
            # e.g. "r *apiKeyRepo" or "r apiKeyRepo"
            recv_parts = recv_raw.strip().split()
            if len(recv_parts) >= 2:
                receiver_type = recv_parts[-1].lstrip("*")

        funcs.append(GoFunc(
            name=match.group("name"),
            receiver_type=receiver_type,
            signature=signature,
            body=body,
            full=full,
        ))

    return funcs


# Test name parsers.
TEST_METHOD_RE = re.compile(
    r"^Test(?P<recv>[A-Z][A-Za-z0-9]*?)_(?P<method>[A-Z][A-Za-z0-9]*)(?:_(?P<scenario>[A-Za-z0-9_]+))?$",
)


def match_source_for_test(test_name: str, src_funcs: list[GoFunc]) -> tuple[GoFunc | None, str | None]:
    """Return (matched source function, scenario hint or None)."""
    # Case 1: Test<ReceiverType>_<Method>[_<Scenario>]
    m = TEST_METHOD_RE.match(test_name)
    if m:
        recv = m.group("recv")
        method = m.group("method")
        scenario = m.group("scenario")
        lower_recv = recv[0].lower() + recv[1:]
        # First prefer exact receiver match (handles apiKeyRepo vs APIKeyRepo).
        for f in src_funcs:
            if f.name == method and f.receiver_type in {recv, lower_recv}:
                return f, scenario
        # Fallback: lowercase-insensitive compare.
        for f in src_funcs:
            if f.name == method and f.receiver_type and f.receiver_type.lower() == recv.lower():
                return f, scenario

    # Case 2: TestFuncName (no underscore) -> free function with same name,
    # case-insensitive first letter (toFoo matches TestToFoo).
    bare = test_name[len("Test"):]
    candidates = [bare, bare[0].lower() + bare[1:]]
    for cand in candidates:
        for f in src_funcs:
            if f.receiver_type is None and f.name == cand:
                return f, None

    return None, None


def iter_test_funcs(funcs: list[GoFunc]) -> list[GoFunc]:
    return [f for f in funcs if f.name.startswith("Test") and f.receiver_type is None]


def build_example(src: GoFunc, scenario: str | None, test: GoFunc) -> dict:
    """Produce an OpenAI-format chat example."""
    user_block = src.full.strip()
    if scenario:
        user_block += f"\n\n// Scenario: {scenario}"
    return {
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": user_block},
            {"role": "assistant", "content": test.full.strip()},
        ],
    }


def extract(source_dir: Path) -> tuple[list[dict], list[str]]:
    """Scan source_dir and return (examples, warnings)."""
    examples: list[dict] = []
    warnings: list[str] = []

    # Build a package-wide view of source functions — test files often exercise
    # converters that live in a sibling .go file with a different basename
    # (e.g. converters_test.go tests helpers from campaign.go).
    package_funcs: list[GoFunc] = []
    for src_path in sorted(source_dir.glob("*.go")):
        if src_path.name.endswith("_test.go"):
            continue
        try:
            src_text = src_path.read_text(encoding="utf-8")
        except OSError as exc:
            warnings.append(f"{src_path}: read failed: {exc}")
            continue
        package_funcs.extend(parse_functions(src_text))

    for test_path in sorted(source_dir.glob("*_test.go")):
        try:
            test_text = test_path.read_text(encoding="utf-8")
        except OSError as exc:
            warnings.append(f"{test_path}: read failed: {exc}")
            continue

        test_funcs = iter_test_funcs(parse_functions(test_text))
        for test_fn in test_funcs:
            matched, scenario = match_source_for_test(test_fn.name, package_funcs)
            if matched is None:
                warnings.append(f"{test_path.name}: {test_fn.name} — no source match")
                continue
            examples.append(build_example(matched, scenario, test_fn))

    return examples, warnings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--source", type=Path, required=True,
        help="directory with *_test.go and sibling *.go files",
    )
    parser.add_argument(
        "--out", type=Path, required=True,
        help="output JSONL path",
    )
    args = parser.parse_args()

    if not args.source.is_dir():
        print(f"error: not a directory: {args.source}", file=sys.stderr)
        return 2

    examples, warnings = extract(args.source)

    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open("w", encoding="utf-8") as fh:
        for ex in examples:
            fh.write(json.dumps(ex, ensure_ascii=False) + "\n")

    print(f"extracted {len(examples)} example(s) -> {args.out}")
    if warnings:
        print(f"warnings ({len(warnings)}):", file=sys.stderr)
        for w in warnings:
            print(f"  {w}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())

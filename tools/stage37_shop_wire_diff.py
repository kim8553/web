#!/usr/bin/env python3
"""Compare two explicitly captured exact-current SHOP_WIRE_OBSERVE windows.

Observational evidence only. An increase in a selector is NOT protocol authority.
Outputs shape/count metadata, never raw lines, addresses, or decoded text.
"""
from __future__ import annotations

import argparse
from collections import Counter
import json
from pathlib import Path
import re
import sys

MAX_LOG_BYTES = 16 * 1024 * 1024
MAX_CANDIDATES = 100
KNOWN_EXCHANGE_SELECTORS = frozenset((64, 69, 79))
OBSERVATION = re.compile(
    r"\bSHOP_WIRE_OBSERVE opcode=0x([0-9a-fA-F]{2}) selector=(-?\d+) "
    r"value_count=(\d+) types=\[([0-9,]*)\](?:\s|$)"
)
ALLOWED_OPCODES = frozenset((0x0A, 0x1E, 0x27))


def parse_window(path: Path) -> tuple[Counter[tuple[int, int, tuple[int, ...]]], int]:
    if not path.is_file():
        raise ValueError(f"input does not exist or is not a file: {path}")
    if path.stat().st_size > MAX_LOG_BYTES:
        raise ValueError(f"input exceeds {MAX_LOG_BYTES} bytes: {path}")
    counts: Counter[tuple[int, int, tuple[int, ...]]] = Counter()
    malformed = 0
    with path.open("r", encoding="utf-8-sig", errors="replace") as stream:
        for line in stream:
            if "SHOP_WIRE_OBSERVE" not in line:
                continue
            found = OBSERVATION.search(line)
            if not found:
                malformed += 1
                continue
            opcode = int(found.group(1), 16)
            selector = int(found.group(2))
            count = int(found.group(3))
            typetext = found.group(4)
            try:
                types = tuple(int(x) for x in typetext.split(",")) if typetext else ()
            except ValueError:
                malformed += 1
                continue
            if (opcode not in ALLOWED_OPCODES or not -2**31 <= selector < 2**31
                    or not 0 <= count <= 65535 or len(types) > 12 or len(types) > count
                    or count > 0 and (not types or types[0] != 2)):
                malformed += 1
                continue
            counts[(opcode, selector, types)] += 1
    return counts, malformed


def compare(baseline: Counter, purchase: Counter, baseline_bad: int, purchase_bad: int) -> dict:
    baseline_total = sum(baseline.values())
    purchase_total = sum(purchase.values())
    if purchase_total == 0:
        raise ValueError("purchase window must contain at least one valid SHOP_WIRE_OBSERVE line")
    candidates = []
    for (opcode, selector, types), after in purchase.items():
        before = baseline.get((opcode, selector, types), 0)
        delta = after - before
        if delta <= 0:
            continue
        candidates.append({
            "opcode": f"0x{opcode:02X}",
            "selector": selector,
            "types": list(types),
            "baseline_count": before,
            "purchase_count": after,
            "count_delta": delta,
            "baseline_share": round(before / baseline_total, 6) if baseline_total else 0.0,
            "purchase_share": round(after / purchase_total, 6),
            "known_mode3_exchange_selector": selector in KNOWN_EXCHANGE_SELECTORS,
            "classification": "OBSERVATIONAL_CANDIDATE_ONLY",
        })
    candidates.sort(key=lambda x: (-x["count_delta"], -x["purchase_share"], x["selector"], x["opcode"], x["types"]))
    return {
        "report_version": 1,
        "authority": "UNVERIFIED_OBSERVATIONAL_DIFFERENCE_ONLY",
        "selector_promotion_allowed": False,
        "handler_enable_allowed": False,
        "baseline_observations": baseline_total,
        "baseline_empty": baseline_total == 0,
        "purchase_observations": purchase_total,
        "malformed_baseline_lines": baseline_bad,
        "malformed_purchase_lines": purchase_bad,
        "candidate_count": len(candidates),
        "candidates": candidates[:MAX_CANDIDATES],
        "truncated_candidates": max(0, len(candidates) - MAX_CANDIDATES),
        "caution": "An increased message count alone cannot prove an ordinary-shop selector; verify exact-current client action, decoded payload, and server flow independently. An empty idle baseline cannot characterize background traffic.",
    }


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--baseline", type=Path, required=True, help="short idle/no-purchase SHOP_WIRE_OBSERVE capture")
    ap.add_argument("--purchase", type=Path, required=True, help="short window containing one manual ordinary NPC purchase click")
    ap.add_argument("--output", type=Path, help="optional JSON report path (no raw log lines)")
    args = ap.parse_args(argv)
    try:
        if args.baseline.resolve() == args.purchase.resolve():
            raise ValueError("baseline and purchase must be separate capture files")
        baseline, bbad = parse_window(args.baseline)
        purchase, pbad = parse_window(args.purchase)
        report = compare(baseline, purchase, bbad, pbad)
        formatted = json.dumps(report, ensure_ascii=False, indent=2) + "\n"
        if args.output is not None:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(formatted, encoding="utf-8")
            print(f"candidate_count={report['candidate_count']} baseline={report['baseline_observations']} purchase={report['purchase_observations']} authority=UNVERIFIED")
        else:
            sys.stdout.write(formatted)
        return 0
    except (OSError, ValueError) as exc:
        print(f"SHOP_WIRE_DIFF_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Parse and compare exact-current Age of Wushu shop-exchange 557 config strings.

This tool is diagnostic-only. It never opens the game process, network, database,
or server. Feed it config_str values captured at the proven current-client
ExchangeItemManager::InitCurExchangeData boundary (RVA 0x00B15560 in the exact
current FxGameLogic.dll) or copied from another authoritative trace.
"""
from __future__ import annotations

import argparse
import json
import pathlib
import sys
from dataclasses import dataclass, asdict
from typing import Iterable

FIELD_NAMES = (
    "Type",
    "AddValue",
    "BindStatus",
    "Item",
    "ShowBind",
    "ExchangeBind",
    "ConditionType",
    "Condition",
    "Condition2",
    "Filters",
    "Prop",
)
INT_FIELDS = {"Type", "BindStatus", "ShowBind", "ExchangeBind", "ConditionType"}
LOG_PREFIX = "JIUYIN_EXCHANGE557_CONFIG="


class ProbeError(ValueError):
    pass


@dataclass(frozen=True)
class Exchange557Record:
    raw: str
    Type: int
    AddValue: str
    BindStatus: int
    Item: str
    ShowBind: int
    ExchangeBind: int
    ConditionType: int
    Condition: str
    Condition2: str
    Filters: str
    Prop: str

    @property
    def preview_visible(self) -> bool:
        # Exact current form_exchange.lua presentation branch.
        return self.BindStatus > 0 and self.ShowBind != 0

    @property
    def preview_bound(self) -> bool | None:
        if not self.preview_visible:
            return None
        return self.ExchangeBind > 0

    def to_dict(self) -> dict:
        out = asdict(self)
        out["preview_visible"] = self.preview_visible
        out["preview_bound"] = self.preview_bound
        return out


def normalize_line(line: str) -> str:
    line = line.strip().rstrip("\r\n")
    if LOG_PREFIX in line:
        line = line.split(LOG_PREFIX, 1)[1].strip()
    if len(line) >= 2 and line[0] == line[-1] and line[0] in {'"', "'"}:
        line = line[1:-1]
    return line.strip()


def parse_config(config: str) -> Exchange557Record:
    raw = normalize_line(config)
    parts = raw.split("|")
    if len(parts) != 11:
        raise ProbeError(f"expected exactly 11 fields, got {len(parts)}: {raw!r}")
    values: dict[str, object] = {name: value for name, value in zip(FIELD_NAMES, parts)}
    for name in INT_FIELDS:
        text = str(values[name]).strip()
        try:
            values[name] = int(text, 10)
        except ValueError as exc:
            raise ProbeError(f"field {name} must be a decimal integer, got {text!r}") from exc
    return Exchange557Record(raw=raw, **values)  # type: ignore[arg-type]


def iter_input_lines(path: str | None, inline: list[str]) -> Iterable[str]:
    for value in inline:
        if value.strip():
            yield value
    if path:
        for line in pathlib.Path(path).read_text(encoding="utf-8-sig").splitlines():
            if LOG_PREFIX in line:
                yield line
            elif line.strip() and not line.lstrip().startswith("#"):
                yield line
    if not inline and not path and not sys.stdin.isatty():
        for line in sys.stdin:
            if line.strip():
                yield line


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--config", action="append", default=[], help="one exact 11-field config_str; repeatable")
    ap.add_argument("--input", help="UTF-8 text/transcript; lines containing JIUYIN_EXCHANGE557_CONFIG= are parsed")
    ap.add_argument("--output", help="write JSON result here instead of stdout")
    ap.add_argument("--require-bind-status", type=int, choices=(0, 1), help="fail if a record has a different BindStatus")
    args = ap.parse_args()

    records: list[Exchange557Record] = []
    errors: list[str] = []
    for idx, line in enumerate(iter_input_lines(args.input, args.config), 1):
        try:
            rec = parse_config(line)
            if args.require_bind_status is not None and rec.BindStatus != args.require_bind_status:
                raise ProbeError(
                    f"BindStatus={rec.BindStatus}, required {args.require_bind_status}"
                )
            records.append(rec)
        except ProbeError as exc:
            errors.append(f"record {idx}: {exc}")

    if not records and not errors:
        errors.append("no config strings supplied")

    result = {
        "schema": list(FIELD_NAMES),
        "record_count": len(records),
        "records": [r.to_dict() for r in records],
        "errors": errors,
        "safe_interpretation": {
            "preview_rule": "visible iff BindStatus>0 and ShowBind!=0; when visible, ExchangeBind>0 displays bound",
            "mutation_authority": False,
            "note": "This tool does not infer persisted item BindStatus from an unbound preview.",
        },
    }
    text = json.dumps(result, ensure_ascii=False, indent=2) + "\n"
    if args.output:
        pathlib.Path(args.output).write_text(text, encoding="utf-8")
    else:
        sys.stdout.write(text)
    return 2 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())

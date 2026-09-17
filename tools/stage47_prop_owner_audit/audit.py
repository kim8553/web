#!/usr/bin/env python3
"""Read-only exact-token census of current checked-out server source.

This finds owner candidates, NOT verified debit, DB persistence or repayment.
No private client bytes, resources, binaries, keys or source lines are read or output.
"""
from __future__ import annotations

import argparse
import pathlib
import re
import subprocess
import sys

UNRESOLVED = (
    'ActivityPartnerPoint', 'ActivityReward_point202504', 'BattleFieldMoney',
    'BiLuoPoint', 'ContributionScore', 'DesertPointBiluo',
    'DesertPointHongchen', 'DesertPointHuangquan', 'DesertPointLow',
    'DongHaiExperience', 'DongYingContributionValue', 'HenggeyuemaValue',
    'HuangQuanPoint', 'JJGGHCCollectPoint', 'JinLanValue', 'MedicalCare',
    'MingContributionValue', 'Point_WangHui', 'RevengeFriendly',
    'SchoolContribute', 'SchoolDanceTotalScore', 'ShmPoint', 'SkillIntegral',
    'SkyHillStar', 'THJobSkillPoint', 'TotalHelpValue', 'WGJobSkillPoint',
    'WSJobSkillPoint', 'XJJobSkillPoint', 'YHJobSkillPoint', 'ZhiBaoPoints',
    'home_res_point', 'qingyipoint', 'tianlunpoint', 'value_lsjianzong',
)
CONTROLS = ('CapitalType1', 'CapitalType2')


def tracked_server_sources(root: pathlib.Path) -> list[pathlib.Path]:
    raw = subprocess.check_output(
        ['git', '-C', str(root), 'ls-files', '-z', '--', 'server'],
        stderr=subprocess.PIPE,
    )
    files = []
    for byte_path in raw.split(b'\x00'):
        if not byte_path:
            continue
        rel = pathlib.Path(byte_path.decode('utf-8', 'surrogateescape'))
        if rel.suffix.lower() not in {'.go', '.sql'}:
            continue
        if (root / rel).is_file():
            files.append(root / rel)
    return sorted(files)


def scan(root: pathlib.Path) -> tuple[list[pathlib.Path], dict[str, list[str]]]:
    names = UNRESOLVED + CONTROLS
    if len(UNRESOLVED) != 35 or len(set(names)) != 37:
        raise ValueError('Stage46 35/37 token inventory drift')
    regexes = {
        name: re.compile(r'(?<![A-Za-z0-9_])' + re.escape(name) + r'(?![A-Za-z0-9_])')
        for name in names
    }
    matches: dict[str, list[str]] = {name: [] for name in names}
    files = tracked_server_sources(root)
    for path in files:
        rel = path.relative_to(root).as_posix()
        # Only tracked Go/SQL text. Do NOT include matched lines or file bytes.
        for number, line in enumerate(path.read_text(encoding='utf-8', errors='replace').splitlines(), 1):
            for name, regex in regexes.items():
                if regex.search(line):
                    matches[name].append(f'{rel}:{number}')
    return files, matches


def report(root: pathlib.Path) -> str:
    files, matches = scan(root)
    head = subprocess.check_output(['git', '-C', str(root), 'rev-parse', 'HEAD'], text=True).strip()
    lines = [
        '# Stage47 current-server exact Prop token census', '',
        f'- Checked-out commit: `{head}`',
        f'- Tracked server Go/SQL files examined: **{len(files)}**',
        '- Scope: case-sensitive whole-token scan of tracked `server/**/*.go` and `server/**/*.sql`.',
        '- Hits are **candidates only**, not proven property owners, persistence or spend authorization.',
        '- Zero hits excludes only the exact spelling in scoped files; aliases, dynamic indices and runtime data are not excluded.',
        '- No private client bytes or matched source lines are included.', '',
        '| Identifier | Matches | File:line locations |',
        '|---|---:|---|',
    ]
    for name in UNRESOLVED + CONTROLS:
        paths = matches[name]
        locations = '<br>'.join(f'`{p}`' for p in paths) if paths else 'NONE IN SCOPED SCAN'
        lines.append(f'| `{name}` | {len(paths)} | {locations} |')
    seen = sum(bool(matches[name]) for name in UNRESOLVED)
    lines.extend(['', f'**Unresolved names with ≥1 exact source-token match: {seen}/35.**',
                  '**End-to-end verified spendable owners: 0/35 from this census alone.**',
                  '**0x4F debit/grant: FAIL-CLOSED.**', ''])
    return '\n'.join(lines)


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument('--root', type=pathlib.Path, default=pathlib.Path(__file__).resolve().parents[2])
    p.add_argument('--output', type=pathlib.Path)
    args = p.parse_args()
    text = report(args.root.resolve())
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(text, encoding='utf-8')
    print(text)
    return 0


if __name__ == '__main__':
    sys.exit(main())

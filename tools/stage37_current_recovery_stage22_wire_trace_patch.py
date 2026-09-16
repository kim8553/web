#!/usr/bin/env python3
"""Replace raw decoded CustomSend logging with passive bounded telemetry."""
from pathlib import Path

path = Path('buildtree/cmd/protocol-probe/main.go')
src = path.read_text(encoding='utf-8')
activity_parser = 'if custom, ok, _ := parseClientActivityCustomMessage(plain); ok && len(custom.Values) > 0 && custom.Values[0].Type == 2 {'
if src.count(activity_parser) != 2:
    raise SystemExit('Stage22: activity parser callsite drift')
activity = src.index(activity_parser)
allowed = src.find('allowed := id == 200', activity)
if allowed < 0 or allowed > activity+350:
    raise SystemExit('Stage22: activity allowed-list drift')
activity_id = src.find('id := custom.Values[0].Int32\n', activity)
if activity_id < 0 or activity_id > allowed:
    raise SystemExit('Stage22: activity selector anchor missing')
insert_at = activity_id + len('id := custom.Values[0].Int32\n')
src = src[:insert_at] + '\t\t\t\ttraceShopWireCustom(custom, conn.RemoteAddr().String())\n' + src[insert_at:]

log_markers = [
    'log.Printf("%s: client activity CustomSend opcode=0x0A msg_id=%d args=[%s]"',
    'log.Printf("%s: unhandled client activity CustomSend opcode=0x0A msg_id=%d args=[%s]"',
    'log.Printf("%s: client CustomSend opcode=0x%02X msg_id=%d args=[%s]"',
]
for index, marker in enumerate(log_markers):
    if src.count(marker) != 1:
        raise SystemExit(f'Stage22: raw log anchor {index} drift')
    log_at = src.index(marker)
    start = src.rfind('arguments := make([]string, 0, len(custom.Values)-1)', 0, log_at)
    if start < 0 or log_at-start > 280:
        raise SystemExit(f'Stage22: argument loop anchor {index} drift')
    start = src.rfind('\n', 0, start)+1
    end = src.find('\n', log_at)
    if end < 0:
        raise SystemExit(f'Stage22: unterminated raw log {index}')
    old = src[start:end+1]
    if 'value.String()' not in old or 'strings.Join(arguments, ", ")' not in old:
        raise SystemExit(f'Stage22: raw value formatter {index} drift')
    indentation = old[:len(old)-len(old.lstrip())]
    replacement = (
        indentation + '// Stage22: observation already emitted once before activity dispatch.\n'
        if index < 2 else
        indentation + 'traceShopWireCustom(custom, conn.RemoteAddr().String())\n'
    )
    src = src[:start] + replacement + src[end+1:]

if src.count('traceShopWireCustom(custom, conn.RemoteAddr().String())') != 2:
    raise SystemExit('Stage22: expected two read-only wire observer callsites')
if 'client CustomSend opcode=0x%02X msg_id=%d args=[%s]' in src or 'client activity CustomSend opcode=0x0A msg_id=%d args=[%s]' in src:
    raise SystemExit('Stage22: unredacted broad CustomSend logging remains')
if 'case 70:' not in src or 'handleShopBuyCustom(' not in src:
    raise SystemExit('Stage22: fail-closed candidate route drift')
path.write_text(src, encoding='utf-8')
print('stage22 activity and ordinary custom logging: two passive observer callsites; three raw argument logs removed')

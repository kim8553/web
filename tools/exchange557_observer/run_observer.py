#!/usr/bin/env python3
"""Attach to the exact current fxgame.exe and record 557 exchange config strings.

The program verifies the running fxgame.exe and sibling fxgamelogic.dll hashes
before attaching. It does not launch the game and does not write game state,
packets, inventory, files in the game directory, or database data.
"""
from __future__ import annotations

import argparse
import ctypes
from ctypes import wintypes
import hashlib
import json
from pathlib import Path
import sys
import time

EXPECTED_EXE_NAME = "fxgame.exe"
EXPECTED_EXE_SHA256 = "c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3"
EXPECTED_DLL_NAME = "fxgamelogic.dll"
EXPECTED_DLL_SHA256 = "16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8"
EXPECTED_FRIDA_VERSION = "17.18.0"
SCHEMA = "nineyin-current-exchange557-observer-v1"
PREFIX = "JIUYIN_EXCHANGE557_CONFIG="


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def windows_process_image_path(pid: int) -> Path:
    if sys.platform != "win32":
        raise RuntimeError("Windows is required for live process verification")
    PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
    kernel32.OpenProcess.restype = wintypes.HANDLE
    kernel32.QueryFullProcessImageNameW.argtypes = [
        wintypes.HANDLE, wintypes.DWORD, wintypes.LPWSTR, ctypes.POINTER(wintypes.DWORD)
    ]
    kernel32.QueryFullProcessImageNameW.restype = wintypes.BOOL
    kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
    kernel32.CloseHandle.restype = wintypes.BOOL
    handle = kernel32.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, False, pid)
    if not handle:
        raise OSError(ctypes.get_last_error(), "OpenProcess failed")
    try:
        buf = ctypes.create_unicode_buffer(32768)
        size = wintypes.DWORD(len(buf))
        if not kernel32.QueryFullProcessImageNameW(handle, 0, buf, ctypes.byref(size)):
            raise OSError(ctypes.get_last_error(), "QueryFullProcessImageNameW failed")
        return Path(buf.value)
    finally:
        kernel32.CloseHandle(handle)


def validate_exact_images(exe: Path) -> tuple[str, Path, str]:
    exe_hash = sha256_file(exe)
    if exe.name.lower() != EXPECTED_EXE_NAME:
        raise RuntimeError(f"running image name {exe.name!r} != {EXPECTED_EXE_NAME!r}")
    if exe_hash.lower() != EXPECTED_EXE_SHA256:
        raise RuntimeError(f"fxgame.exe SHA256 mismatch: {exe_hash}")
    dll = exe.parent / EXPECTED_DLL_NAME
    if not dll.is_file():
        raise RuntimeError(f"current FxGameLogic.dll not found beside running client: {dll}")
    dll_hash = sha256_file(dll)
    if dll_hash.lower() != EXPECTED_DLL_SHA256:
        raise RuntimeError(f"FxGameLogic.dll SHA256 mismatch: {dll_hash}")
    return exe_hash.lower(), dll, dll_hash.lower()


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--pid", type=int, help="fxgame.exe PID; omit if exactly one is running")
    ap.add_argument("--out-dir", type=Path, default=Path("exchange557_capture"))
    args = ap.parse_args()

    try:
        import frida  # type: ignore
    except Exception as exc:
        print(f"ERROR: frida is required: {exc}", file=sys.stderr)
        return 2
    if getattr(frida, "__version__", "unknown") != EXPECTED_FRIDA_VERSION:
        print(
            f"ERROR: requires frida=={EXPECTED_FRIDA_VERSION}; found {getattr(frida, '__version__', 'unknown')}",
            file=sys.stderr,
        )
        return 3

    device = frida.get_local_device()
    pid = args.pid
    if pid is None:
        matches = [p for p in device.enumerate_processes() if p.name.lower() == EXPECTED_EXE_NAME]
        if len(matches) != 1:
            print(f"ERROR: expected exactly one {EXPECTED_EXE_NAME}; found {len(matches)}. Pass --pid.", file=sys.stderr)
            return 4
        pid = matches[0].pid

    try:
        exe = windows_process_image_path(pid)
        exe_hash, dll, dll_hash = validate_exact_images(exe)
    except Exception as exc:
        print(f"ERROR: exact-image precheck failed: {exc}", file=sys.stderr)
        return 5

    stamp = time.strftime("%Y%m%d_%H%M%S")
    out_dir = args.out_dir.resolve()
    out_dir.mkdir(parents=True, exist_ok=True)
    jsonl = out_dir / f"exchange557_{stamp}.jsonl"
    configs = out_dir / f"exchange557_{stamp}.txt"
    agent = Path(__file__).with_name("agent.js").read_text(encoding="utf-8")
    agent_sha = hashlib.sha256(agent.encode("utf-8")).hexdigest()

    print(f"PID              : {pid}")
    print(f"fxgame.exe       : {exe}")
    print(f"fxgame SHA256    : {exe_hash}")
    print(f"FxGameLogic.dll  : {dll}")
    print(f"FxGameLogic SHA  : {dll_hash}")
    print(f"JSONL            : {jsonl}")
    print(f"Config strings   : {configs}")

    session = None
    script = None
    stopped = False
    ready = False
    with jsonl.open("x", encoding="utf-8", buffering=1) as jf, configs.open("x", encoding="utf-8", buffering=1) as cf:
        def write(rec: dict) -> None:
            rec.setdefault("timestamp_unix_ns", time.time_ns())
            jf.write(json.dumps(rec, ensure_ascii=False, separators=(",", ":")) + "\n")

        write({
            "schema": SCHEMA,
            "event": "HOST_VERIFIED",
            "pid": pid,
            "fxgame_path": str(exe),
            "fxgame_sha256": exe_hash,
            "fxgamelogic_path": str(dll),
            "fxgamelogic_sha256": dll_hash,
            "frida_version": EXPECTED_FRIDA_VERSION,
            "agent_sha256": agent_sha,
        })

        try:
            session = device.attach(pid)

            def on_detached(reason, crash=None):
                nonlocal stopped
                write({"schema": SCHEMA, "event": "FRIDA_SESSION_DETACHED", "reason": str(reason), "crash": None if crash is None else str(crash)})
                stopped = True

            session.on("detached", on_detached)
            script = session.create_script(agent)

            def on_message(message, data):
                nonlocal stopped, ready
                if message.get("type") == "send":
                    wrapper = message.get("payload") or {}
                    if wrapper.get("kind") == "event" and isinstance(wrapper.get("payload"), dict):
                        rec = dict(wrapper["payload"])
                        write(rec)
                        ev = rec.get("event")
                        if ev == "OBSERVER_READY":
                            ready = True
                        elif ev == "EXCHANGE557_CONFIG":
                            raw = str(rec.get("raw", ""))
                            if len(raw.split("|")) == 11:
                                cf.write(PREFIX + raw + "\n")
                                print(PREFIX + raw)
                            else:
                                write({"schema": SCHEMA, "event": "HOST_VALIDATION_ERROR", "error": "agent emitted non-11-field config"})
                        elif ev == "EXCHANGE557_ANOMALY":
                            print(json.dumps(rec, ensure_ascii=False), file=sys.stderr)
                        return
                write({"schema": SCHEMA, "event": "FRIDA_MESSAGE", "message": message, "binary_len": None if data is None else len(data)})
                if message.get("type") == "error":
                    stopped = True

            script.on("message", on_message)
            script.load()
            deadline = time.time() + 3.0
            while not ready and not stopped and time.time() < deadline:
                time.sleep(0.05)
            if not ready:
                raise RuntimeError("observer did not reach OBSERVER_READY; see JSONL for agent error")

            print("Observer ready. Open one BindStatus=1 exchange form in game, then press Enter here.")
            try:
                input()
            except (EOFError, KeyboardInterrupt):
                pass
        except Exception as exc:
            write({"schema": SCHEMA, "event": "OBSERVER_FAILED", "error": repr(exc)})
            print(f"ERROR: {exc}", file=sys.stderr)
            return 6
        finally:
            if script is not None:
                try:
                    script.unload()
                except Exception as exc:
                    write({"schema": SCHEMA, "event": "SCRIPT_UNLOAD_ERROR", "error": repr(exc)})
            if session is not None:
                try:
                    session.detach()
                except Exception as exc:
                    write({"schema": SCHEMA, "event": "SESSION_DETACH_ERROR", "error": repr(exc)})
            write({"schema": SCHEMA, "event": "HOST_DETACHED", "pid": pid})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

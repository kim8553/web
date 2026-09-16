import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "tools" / "stage37_shop_wire_diff.py"
spec = importlib.util.spec_from_file_location("stage37_shop_wire_diff", SCRIPT)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)


def line(opcode, selector, types, detail=""):
    return (f"2026-09-16 x: SHOP_WIRE_OBSERVE opcode=0x{opcode:02X} "
            f"selector={selector} value_count={len(types)} types=[{','.join(str(x) for x in types)}]"
            f"{detail}\n")


class ShopWireDiffTests(unittest.TestCase):
    def test_candidates_never_promoted_and_mode3_marked(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory) / "baseline.log"
            after = Path(directory) / "purchase.log"
            base.write_text(line(0x1e, 10, [2]) * 2 + line(0x27, 64, [2, 6]), encoding="utf-8")
            after.write_text(line(0x1e, 10, [2]) * 2 + line(0x1e, 70, [2, 6, 2, 2, 2, 2], " detail=[1:shopid=Shop_ZH_001]") * 3 + line(0x27, 64, [2, 6]) * 2, encoding="utf-8")
            b, bb = mod.parse_window(base)
            p, pb = mod.parse_window(after)
            report = mod.compare(b, p, bb, pb)
            self.assertEqual((report["baseline_observations"], report["purchase_observations"]), (3, 7))
            self.assertFalse(report["selector_promotion_allowed"])
            self.assertFalse(report["handler_enable_allowed"])
            self.assertEqual(report["candidates"][0]["selector"], 70)
            self.assertEqual(report["candidates"][0]["count_delta"], 3)
            self.assertTrue(next(x for x in report["candidates"] if x["selector"] == 64)["known_mode3_exchange_selector"])
            self.assertNotIn("Shop_ZH_001", json.dumps(report))

    def test_invalid_lines_counted_without_reprinting_private_payload(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "capture.log"
            path.write_text("private: SHOP_WIRE_OBSERVE bad_password=secret\n" + line(0x1e, 1, [2]) + line(0x05, 2, [2]) + line(0x27, 3, [6]), encoding="utf-8")
            counts, bad = mod.parse_window(path)
            self.assertEqual(sum(counts.values()), 1)
            self.assertEqual(bad, 3)

    def test_empty_file_and_same_capture_fail_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "empty.log"
            path.write_text("unrelated\n", encoding="utf-8")
            self.assertEqual(subprocess.run([sys.executable, str(SCRIPT), "--baseline", str(path), "--purchase", str(path)], capture_output=True).returncode, 2)
            other = Path(directory) / "other.log"
            other.write_text(line(0x1e, 2, [2]), encoding="utf-8")
            result = subprocess.run([sys.executable, str(SCRIPT), "--baseline", str(path), "--purchase", str(other)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(json.loads(result.stdout)["baseline_empty"])

    def test_cli_json_output_has_no_raw_text(self):
        with tempfile.TemporaryDirectory() as directory:
            b = Path(directory) / "b.log"
            p = Path(directory) / "p.log"
            output = Path(directory) / "report.json"
            b.write_text(line(0x0a, 200, [2, 6]), encoding="utf-8")
            p.write_text(line(0x0a, 200, [2, 6]) + line(0x27, 900, [2, 7], " detail=[1:textlen=12,sha256_8=12345678]"), encoding="utf-8")
            result = subprocess.run([sys.executable, str(SCRIPT), "--baseline", str(b), "--purchase", str(p), "--output", str(output)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            report = json.loads(output.read_text(encoding="utf-8"))
            self.assertEqual(report["candidates"][0]["selector"], 900)
            self.assertEqual(report["candidates"][0]["opcode"], "0x27")
            self.assertNotIn("12345678", output.read_text(encoding="utf-8"))

    def test_overlarge_input_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "large.log"
            with path.open("wb") as file:
                file.truncate(mod.MAX_LOG_BYTES + 1)
            with self.assertRaisesRegex(ValueError, "exceeds"):
                mod.parse_window(path)


if __name__ == "__main__":
    unittest.main()

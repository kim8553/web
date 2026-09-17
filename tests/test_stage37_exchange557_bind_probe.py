import importlib.util
import pathlib
import sys
import unittest

P = pathlib.Path(__file__).resolve().parents[1] / "tools" / "stage37_exchange557_bind_probe.py"
spec = importlib.util.spec_from_file_location("stage37_exchange557_bind_probe", P)
probe = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = probe
assert spec.loader is not None
spec.loader.exec_module(probe)


class ProbeTests(unittest.TestCase):
    def test_exact_11_fields_bound_preview(self):
        r = probe.parse_config("0||1|tab_card_001,1|1|1|0||||")
        self.assertEqual(r.BindStatus, 1)
        self.assertTrue(r.preview_visible)
        self.assertTrue(r.preview_bound)

    def test_exact_11_fields_unbound_preview(self):
        r = probe.parse_config("0||1|tab_card_001,1|1|0|0||||")
        self.assertTrue(r.preview_visible)
        self.assertFalse(r.preview_bound)

    def test_showbind_zero_hides_preview(self):
        r = probe.parse_config("0||1|tab_card_001,1|0|1|0||||")
        self.assertFalse(r.preview_visible)
        self.assertIsNone(r.preview_bound)

    def test_bindstatus_zero_hides_preview(self):
        r = probe.parse_config("0||0|tab_card_001,1|1|1|0||||")
        self.assertFalse(r.preview_visible)
        self.assertIsNone(r.preview_bound)

    def test_prefix_is_accepted(self):
        r = probe.parse_config("[dbg] JIUYIN_EXCHANGE557_CONFIG=0||1|x,1|1|0|0||||")
        self.assertEqual(r.Item, "x,1")

    def test_wrong_field_count_fails(self):
        with self.assertRaises(probe.ProbeError):
            probe.parse_config("0|1|2")

    def test_integer_field_validation(self):
        with self.assertRaises(probe.ProbeError):
            probe.parse_config("x||1|x,1|1|0|0||||")


if __name__ == "__main__":
    unittest.main()

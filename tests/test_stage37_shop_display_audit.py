import importlib.util
from pathlib import Path
import unittest

SPEC = importlib.util.spec_from_file_location('audit', Path(__file__).resolve().parents[1] / 'tools' / 'stage37_shop_display_audit.py')
audit = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(audit)


class DisplayAuditTests(unittest.TestCase):
    def test_mode3_only_shop_has_no_add_frames(self):
        report = audit.scan(b'[Shop_A]\nType=1\n0=item,1,3,0,0,0,0,100\n', 'Shop_A')
        self.assertEqual(report['requested_shop']['status'], 'NO_ITEM_ADD_FRAMES_FROM_MODE_FILTER')
        self.assertEqual(report['section_categories']['no_supported_price_mode'], 1)
        self.assertFalse(report['selector_verified'])

    def test_ordinary_mixed_and_empty(self):
        raw = b'[Shop_A]\n0=item,1,1,15,0,0,0,0\n0=another,1,3,0,0,0,1,200\n[Shop_B]\nType=1\n'
        report = audit.scan(raw, 'Shop_A')
        self.assertEqual(report['requested_shop']['eligible_rows'], 1)
        self.assertEqual(report['section_categories']['no_item_rows'], 1)
        self.assertEqual(report['parsed_price_mode_rows'], {1: 1, 3: 1})
        self.assertEqual(audit.scan(raw, 'Shop_C')['requested_shop']['status'], 'SECTION_NOT_FOUND')

    def test_parser_error_and_view_capacity_are_distinct(self):
        raw = b'[bad]\n0=item,1,1,15,0,0,NOT_INT\n[large]\n0=item,1,2,0,0,0,100\n'
        report = audit.scan(raw, 'large')
        self.assertEqual(report['section_categories']['parser_error'], 1)
        self.assertEqual(report['eligible_rows_outside_declared_view_capacity_100'], 1)
        self.assertEqual(report['requested_shop']['status'], 'ITEM_ADD_FRAMES_POSSIBLE_NOT_LIVE_VERIFIED')
        self.assertTrue(report['view_capacity_mismatch_is_client_effect_unverified'])

    def test_large_page_triggers_index_guard(self):
        result = audit.scan(b'[a]\n200=item,1,1,1,0,0,0\n', 'a')
        self.assertEqual(result['section_categories']['invalid_view_index_candidate'], 1)
        self.assertEqual(result['eligible_rows_with_invalid_view_index'], 1)


if __name__ == '__main__':
    unittest.main()

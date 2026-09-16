import tempfile
import unittest
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'tools'))
from stage35_shop_trace_audit import read_logs

class ShopTraceAuditTests(unittest.TestCase):
    def audit(self, content):
        with tempfile.TemporaryDirectory() as temp:
            file = Path(temp, 'test.log')
            file.write_text(content, encoding='utf-8')
            return read_logs([file])

    def test_exact_missing_catalog_and_no_suffix_guess(self):
        report = self.audit('2026/09/16 NPC shop menu diagnostic object=10 npc_config="npcX" shop="Shop_GB_Yishiting" source="template" catalog_error="shop has no section" authored=0 ordinary=0 exchange=0\n')
        shop = report['shops'][0]
        self.assertEqual(shop['classification'], 'exact_catalog_error_observed')
        self.assertEqual(shop['shop_id'], 'Shop_GB_Yishiting')
        self.assertEqual(shop['npc_config_ids'], ['npcX'])

    def test_exchange_only_menu_does_not_claim_render(self):
        report = self.audit('NPC shop menu diagnostic object=21 npc_config="npcE" shop="Exchange" catalog_error="" authored=3 ordinary=0 exchange=3\n')
        self.assertEqual(report['shops'][0]['classification'], 'menu_catalog_has_zero_ordinary_rows_display_unverified')

    def test_server_frames_not_client_render(self):
        report = self.audit('shop service selected npc_config="npc1" shop="ShopA" service_source="template"\nshop display catalog shop="ShopA" rows=2 ordinary=2\nshop display frames shop="ShopA" create_view=sent item_add_sent=2 exchange_skipped=0 unsupported_skipped=0 client_render=unverified\n')
        self.assertEqual(report['shops'][0]['classification'], 'server_frame_write_completion_logged_client_display_unverified')
        self.assertEqual(report['shops'][0]['frame_item_add_counts'], [2])

    def test_preflight_rejected(self):
        report = self.audit('shop display preflight rejected shop="S" rows=2 reason=duplicate client_view_created=false\n')
        self.assertEqual(report['shops'][0]['classification'], 'preflight_rejection_observed')

    def test_empty_unrelated_and_unattributed(self):
        report = self.audit('server started\nshop display frames create_view=sent item_add_sent=1\n')
        self.assertEqual(report['shops'], [])
        self.assertEqual(report['unattributed_event_lines'], 1)

    def test_no_raw_personal_identifiers_in_result(self):
        report = self.audit('10.0.0.1:1234: NPC shop menu diagnostic object=999 owner=123 scene="city05" npc_config="npc" shop="S" catalog_error="" ordinary=1 exchange=0\n')
        self.assertNotIn('10.0.0.1', str(report))
        self.assertNotIn('owner', str(report))

if __name__ == '__main__':
    unittest.main()

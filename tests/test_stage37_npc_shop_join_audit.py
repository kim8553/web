import importlib.util
import pathlib
import unittest

MODULE = pathlib.Path(__file__).resolve().parents[1] / 'tools' / 'stage37_npc_shop_join_audit.py'
spec = importlib.util.spec_from_file_location('stage37_npc_shop_join_audit', MODULE)
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)


def table(records):
    header = b'NPC\tOBJ\tSHOP\nSTRING\tSTRING\tSTRING\nID\tObjectType\tShopID\n\t\t\n\t\t\n'
    return header + b''.join(b'\t'.join(fields) + b'\n' for fields in records)


SHOP = (b'[Normal]\n0=item_a,1,1,50,0,0,0,0,,\n'
        b'[Exchange]\n0=item_b,1,3,50,0,0,0,0,,\n'
        b'[Empty]\nType=1\n')


class TestJoin(unittest.TestCase):
    def test_classification_and_explicit_npc(self):
        data = table([(b'n1', b'x', b'Normal'), (b'n2', b'x', b'Exchange'),
                      (b'n3', b'x', b'Empty'), (b'n4', b'x', b'Missing'),
                      (b'n5', b'x', b'')])
        result = audit.audit(data, SHOP, 'n2')
        self.assertEqual(result['npc_rows_with_shop_id'], 4)
        self.assertEqual(result['distinct_shop_ids_referenced'], 4)
        self.assertEqual(result['npc_shop_classification'], {
            'NORMAL_MODE_ROWS_PRESENT_NOT_LIVE_VERIFIED': 1,
            'NO_NORMAL_MODE_ITEM_ADD_FRAMES': 1,
            'NO_ITEM_ROWS': 1,
            'SECTION_NOT_FOUND': 1})
        self.assertEqual(result['requested_npc']['shop_id'], 'Exchange')
        self.assertEqual(result['requested_npc']['price_mode_counts'], {3: 1})
        self.assertFalse(result['live_e2e_verified'])

    def test_conflicting_duplicate_npc_rejected(self):
        with self.assertRaisesRegex(ValueError, 'conflicting duplicate NPC ID'):
            audit.audit(table([(b'n1', b'x', b'Normal'), (b'n1', b'x', b'Empty')]), SHOP)

    def test_blank_and_identical_duplicate_rows_not_invented(self):
        result = audit.audit(table([(b'', b'present', b''), (b'n1', b'x', b'Normal'),
                                    (b'n1', b'x', b'Normal')]), SHOP)
        self.assertEqual(result['npc_blank_id_lines_excluded'], 1)
        self.assertEqual(result['npc_identical_duplicate_id_lines_excluded'], 1)
        self.assertEqual(result['npc_unique_nonblank_ids'], 1)

    def test_shop_without_npc_id_rejected(self):
        with self.assertRaisesRegex(ValueError, 'ShopID without NPC ID'):
            audit.audit(table([(b'', b'x', b'Normal')]), SHOP)

    def test_corrupted_width_rejected(self):
        with self.assertRaisesRegex(ValueError, 'column'):
            audit.audit(table([(b'n1', b'x')]), SHOP)

    def test_duplicate_shop_section_rejected(self):
        with self.assertRaisesRegex(ValueError, 'duplicate shop section'):
            audit.audit(table([(b'n1', b'x', b'Normal')]), SHOP+b'[Normal]\n')

    def test_invalid_row_blocks_normal_claim(self):
        invalid = b'[Normal]\n0=item_a,1,1,50,0,0,0,0,,\n1=bad\n'
        result = audit.audit(table([(b'n1', b'x', b'Normal')]), invalid, 'n1')
        self.assertEqual(result['requested_npc']['status'], 'SHOP_ROW_PARSE_ERROR')

    def test_missing_npc_not_assumed(self):
        result = audit.audit(table([(b'n1', b'x', b'Normal')]), SHOP, 'n_other')
        self.assertEqual(result['requested_npc']['status'], 'NPC_ID_NOT_IN_THIS_TABLE')


if __name__ == '__main__':
    unittest.main()

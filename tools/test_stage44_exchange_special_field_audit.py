"""Only synthetic fixtures: private current INI bytes must not enter CI."""
import hashlib
import tempfile
import unittest
from pathlib import Path

from stage44_exchange_special_field_audit import analyze, mode3_refs, parse_ini, read_exact


class SpecialFieldTests(unittest.TestCase):
    def test_cross_tab_count_is_definition_not_listing_count(self):
        exchange = '''[10]\nType=1\nAddValue=10000000\n[11]\nType=2\nAddValue=2000\nItem=item_a,1\n[12]\nType=3\nProp=SchoolContribute,10\n[13]\nType=3\nItem=item_b,2\nProp=WGJobSkillPoint,20\n[14]\nBindStatus=1\n'''
        shop = '''[ShopTest]\n0=result_a,1,3,0,0,1,0,10,0\n1=result_b,1,3,0,0,1,0,11,0\n2=result_c,1,3,0,0,1,0,12,0\n3=result_d,1,3,0,0,1,0,13,0\n4=result_e,1,3,0,0,1,0,13,0\n5=result_f,1,3,0,0,1,0,14,0\n6=result_g,1,3,0,0,1,0,0,0\n7=result_h,1,2,0,0,1,0,999,0\n'''
        result = analyze(shop, exchange)
        self.assertEqual(result['mode3_rows'], 7)
        self.assertEqual(result['mode3_zero_exchange'], 1)
        self.assertEqual(result['mode3_referenced_rows'], 6)
        self.assertEqual(result['referenced_exchange_definitions'], 5)
        self.assertEqual(result['by_type_definitions']['3'], {'Item+Prop': 1, 'Prop': 1})
        self.assertEqual(result['by_type_shop_rows']['3'], {'Item+Prop': 2, 'Prop': 1})
        self.assertEqual(result['nonzero_bindstatus'], {'definitions': 1, 'shop_rows': 1})
        self.assertEqual(result['type3_item_count'], 1)
        self.assertEqual(result['addvalue_by_type_and_authored_text'], [
            {'type': '1', 'value': '10000000', 'definitions': 1},
            {'type': '2', 'value': '2000', 'definitions': 1},
        ])
        self.assertEqual(result['prop_names_by_definition_count'], {'SchoolContribute': 1, 'WGJobSkillPoint': 1})

    def test_missing_authority_is_rejected(self):
        with self.assertRaisesRegex(ValueError, 'missing ExchangeItem'):
            analyze('[shop]\n1=item,1,3,0,0,1,0,999,0\n', '[10]\nItem=a,1\n')

    def test_duplicate_exchange_section_rejected(self):
        with self.assertRaisesRegex(ValueError, 'duplicate/empty section'):
            parse_ini('[10]\nItem=a,1\n[10]\nItem=b,2\n')

    def test_duplicate_exchange_key_rejected(self):
        with self.assertRaisesRegex(ValueError, 'duplicate/empty key'):
            parse_ini('[10]\nItem=a,1\nItem=b,2\n')

    def test_malformed_mode3_exchange_rejected(self):
        with self.assertRaisesRegex(ValueError, 'not numeric'):
            mode3_refs('[Shop]\n0=item,1,3,0,0,1,0,notanid,0\n')

    def test_different_file_digest_rejected_without_parsing(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'shop.ini'
            path.write_bytes(b'[wrong]\n')
            with self.assertRaisesRegex(ValueError, 'SHA256 mismatch'):
                read_exact(path, '0' * 64)

    def test_file_digest_correct_is_accepted(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'test.ini'
            payload = b'[ok]\nItem=x,2\n'
            path.write_bytes(payload)
            self.assertEqual(read_exact(path, hashlib.sha256(payload).hexdigest()), payload.decode('latin-1'))

    def test_unknown_type_not_silently_reclassified(self):
        result = analyze('[S]\n0=item,1,3,0,0,1,0,7,0\n', '[7]\nType=98\nItem=i,1\n')
        self.assertEqual(result['unknown_type_ids'], ['7'])
        self.assertEqual(result['by_type_definitions']['98'], {'Item': 1})


if __name__ == '__main__':
    unittest.main()

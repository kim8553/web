"""Synthetic-only tests; contains no game resource data."""
import unittest

from stage43_skill_new_preflight import audit


class SkillNewPreflightTests(unittest.TestCase):
    def test_two_supported_scripts_and_no_identifiers_disclosed(self):
        data = b"[sample_one]\r\nscript=SkillNormal\r\nStaticData=123\r\n[sample_two]\nscript=SkillLock\nstaticdata=456\n"
        report = audit(data)
        self.assertEqual(report["unique_sections"], 2)
        self.assertEqual(report["go_skill_index_candidates"], 2)
        self.assertEqual(report["missing_staticdata_in_candidates"], 0)
        self.assertTrue(report["scanner_preflight_pass"])
        self.assertNotIn("sample_one", str(report))

    def test_non_utf8_bytes_are_not_implicitly_rejected(self):
        data = b"[\xc1\xd1]\nscript=SkillNormal\nStaticData=1\n"
        report = audit(data)
        self.assertFalse(report["utf8_valid"])
        self.assertTrue(report["scanner_preflight_pass"])
        self.assertEqual(report["go_skill_index_candidates"], 1)

    def test_duplicate_header_is_reported_and_missing_static_counted(self):
        data = b"[same]\nscript=SkillNormal\n[same]\nscript=SkillLock\n"
        report = audit(data)
        self.assertEqual(report["duplicate_section_names"], 1)
        self.assertEqual(report["missing_staticdata_in_candidates"], 1)
        self.assertEqual(report["go_skill_index_candidates"], 1)

    def test_nul_is_blocked(self):
        report = audit(b"[one]\nscript=SkillNormal\nStaticData=1\n\x00")
        self.assertFalse(report["scanner_preflight_pass"])
        self.assertIn("nul_bytes", report["scanner_blockers"])

    def test_empty_input_is_blocked(self):
        self.assertFalse(audit(b"")["scanner_preflight_pass"])


if __name__ == "__main__":
    unittest.main()

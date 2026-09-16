"""Synthetic-only checks: no private client or game data is used."""
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

TOOL = Path(__file__).with_name('stage44_resource_loader_audit.go')
SKILLS = ('skill_static.ini', 'skill_normal_varprop.ini', 'skill_lock_varprop.ini',
          'skill_consume.ini', 'damage_calculate.ini', 'attack_hitshape.ini',
          'attack_targetshape.ini', 'buff_new.ini', 'buff_static.ini', 'buff_varprop.ini')
ACTIONS = ('zhaoshi_player.ini', 'zhaoshi_player_2.ini',
           'zhaoshi_player_dodge.ini', 'zhaoshi_player_parry.ini', 'zhaoshi_clone.ini')

class ResourceAuditTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.skill = self.root / 'skill'
        self.action = self.root / 'action'
        self.skill.mkdir()
        self.action.mkdir()
        self.new = self.root / 'skill_new.ini'
        self.new.write_text('[A]\nscript=SkillNormal\nStaticData=1\n[B]\nscript=SkillLock\nStaticData=2\n')
        for name in SKILLS:
            (self.skill / name).write_text('[dummy]\nValue=1\n')
        (self.skill / 'skill_static.ini').write_text('[1]\nMinVarPropNo=1\nMaxVarPropNo=1\n[2]\nMinVarPropNo=2\nMaxVarPropNo=2\n')
        (self.skill / 'skill_normal_varprop.ini').write_text('[1]\nLevel=1\n')
        (self.skill / 'skill_lock_varprop.ini').write_text('[2]\nLevel=1\n')
        for name in ACTIONS:
            (self.action / name).write_text('[placeholder]\nkey=value\n')
        (self.action / 'zhaoshi_player.ini').write_text('[A]\n0=pose;action;30;0;other\n')
        (self.action / 'zhaoshi_clone.ini').write_text('[A]\nkey=invalid\n[B]\n0=pose;action;30;0;other\n')
        self.shop = self.root / 'shop.ini'
        self.legacy = self.root / 'legacy.ini'
        self.shop.write_bytes(b'[sample]\n')
        self.legacy.write_bytes(b'[sample]\n')

    def invoke(self, *extra):
        return subprocess.run(['go', 'run', str(TOOL), '--skill-new', str(self.new),
                               '--skill-root', str(self.skill), '--action-root', str(self.action),
                               *extra], text=True, capture_output=True, check=False)

    def test_all_16_clone_fallback_and_equal_shop(self):
        p = self.invoke('--shop', str(self.shop), '--legacy-shop', str(self.legacy))
        self.assertEqual(p.returncode, 0, p.stderr)
        v = json.loads(p.stdout)
        self.assertEqual(v['inputs_read'], 16)
        self.assertEqual(v['indexed_supported_scripts'], 2)
        self.assertEqual(v['level_one_preflight'], 2)
        self.assertTrue(v['shop_legacy_identical'])
        self.assertNotIn('"A"', p.stdout)
        self.assertNotIn('"B"', p.stdout)

    def test_missing_static_is_not_usable(self):
        (self.skill / 'skill_static.ini').write_text('[1]\nMinVarPropNo=1\nMaxVarPropNo=1\n')
        p = self.invoke()
        self.assertEqual(p.returncode, 0, p.stderr)
        v = json.loads(p.stdout)
        self.assertEqual(v['level_one_preflight'], 1)
        self.assertEqual(v['reason_counts']['missing_static'], 1)
        self.assertNotIn('shop_legacy_identical', v)

    def test_missing_action_file_fails_without_fake_data(self):
        (self.action / 'zhaoshi_player_2.ini').unlink()
        p = self.invoke()
        self.assertNotEqual(p.returncode, 0)
        self.assertEqual(p.stdout.strip(), '')

    def test_shop_mismatch_and_missing_comparison_pair(self):
        self.legacy.write_bytes(b'different')
        p = self.invoke('--shop', str(self.shop), '--legacy-shop', str(self.legacy))
        self.assertEqual(p.returncode, 0, p.stderr)
        self.assertFalse(json.loads(p.stdout)['shop_legacy_identical'])
        p = self.invoke('--shop', str(self.shop))
        self.assertNotEqual(p.returncode, 0)
        self.assertEqual(p.stdout.strip(), '')

if __name__ == '__main__':
    unittest.main()

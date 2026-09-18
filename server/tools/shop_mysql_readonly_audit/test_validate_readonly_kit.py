"""Strictly offline tests; never connect to any MySQL server."""
import unittest
from validate_readonly_kit import validate_sql


class ReadOnlySQLValidationTests(unittest.TestCase):
    def test_plain_select(self):
        self.assertEqual(validate_sql('-- harmless\nSELECT DATABASE();'), 1)

    def test_disallow_writes(self):
        for text in ['DELETE FROM role_currency;', 'CREATE TABLE t(x INT);',
                     'SELECT 1 INTO OUTFILE "/tmp/leak";',
                     'SELECT role_id FROM roles FOR UPDATE;',
                     'SELECT GET_LOCK("a",1);', 'SELECT LOAD_FILE("/etc/passwd");']:
            with self.subTest(text=text), self.assertRaises(AssertionError):
                validate_sql(text)

    def test_no_empty_file(self):
        with self.assertRaises(AssertionError):
            validate_sql('-- nothing here')

    def test_multiple_selects(self):
        self.assertEqual(validate_sql('SELECT DATABASE(); SELECT 1;'), 2)


if __name__ == '__main__':
    unittest.main()

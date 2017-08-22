from __future__ import with_statement

from vcs.utils.filesize import filesizeformat
from vcs.utils.compat import unittest


class TestFilesizeformat(unittest.TestCase):

    def test_bytes(self):
        self.assertEqual(filesizeformat(10), '10 B')

    def test_kilobytes(self):
        self.assertEqual(filesizeformat(1024 * 2), '2 KB')

    def test_megabytes(self):
        self.assertEqual(filesizeformat(1024 * 1024 * 2.3), '2.3 MB')

    def test_gigabytes(self):
        self.assertEqual(filesizeformat(1024 * 1024 * 1024 * 12.92), '12.92 GB')

    def test_that_function_respects_sep_paramtere(self):
        self.assertEqual(filesizeformat(1, ''), '1B')


if __name__ == '__main__':
    unittest.main()

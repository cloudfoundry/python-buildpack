from __future__ import with_statement

import os
import tarfile
import zipfile
import datetime
import tempfile
import StringIO
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS
from vcs.exceptions import VCSError
from vcs.nodes import FileNode
from vcs.utils.compat import unittest


class ArchivesTestCaseMixin(BackendTestMixin):

    @classmethod
    def _get_commits(cls):
        start_date = datetime.datetime(2010, 1, 1, 20)
        for x in xrange(5):
            yield {
                'message': 'Commit %d' % x,
                'author': 'Joe Doe <joe.doe@example.com>',
                'date': start_date + datetime.timedelta(hours=12 * x),
                'added': [
                    FileNode('%d/file_%d.txt' % (x, x),
                        content='Foobar %d' % x),
                ],
            }

    def test_archive_zip(self):
        path = tempfile.mkstemp()[1]
        with open(path, 'wb') as f:
            self.tip.fill_archive(stream=f, kind='zip', prefix='repo')
        out = zipfile.ZipFile(path)

        for x in xrange(5):
            node_path = '%d/file_%d.txt' % (x, x)
            decompressed = StringIO.StringIO()
            decompressed.write(out.read('repo/' + node_path))
            self.assertEqual(
                decompressed.getvalue(),
                self.tip.get_node(node_path).content)

    def test_archive_tgz(self):
        path = tempfile.mkstemp()[1]
        with open(path, 'wb') as f:
            self.tip.fill_archive(stream=f, kind='tgz', prefix='repo')
        outdir = tempfile.mkdtemp()

        outfile = tarfile.open(path, 'r|gz')
        outfile.extractall(outdir)

        for x in xrange(5):
            node_path = '%d/file_%d.txt' % (x, x)
            self.assertEqual(
                open(os.path.join(outdir, 'repo/' + node_path)).read(),
                self.tip.get_node(node_path).content)

    def test_archive_tbz2(self):
        path = tempfile.mkstemp()[1]
        with open(path, 'w+b') as f:
            self.tip.fill_archive(stream=f, kind='tbz2', prefix='repo')
        outdir = tempfile.mkdtemp()

        outfile = tarfile.open(path, 'r|bz2')
        outfile.extractall(outdir)

        for x in xrange(5):
            node_path = '%d/file_%d.txt' % (x, x)
            self.assertEqual(
                open(os.path.join(outdir, 'repo/' + node_path)).read(),
                self.tip.get_node(node_path).content)

    def test_archive_default_stream(self):
        tmppath = tempfile.mkstemp()[1]
        with open(tmppath, 'w') as stream:
            self.tip.fill_archive(stream=stream)
        mystream = StringIO.StringIO()
        self.tip.fill_archive(stream=mystream)
        mystream.seek(0)
        with open(tmppath, 'r') as f:
            self.assertEqual(f.read(), mystream.read())

    def test_archive_wrong_kind(self):
        with self.assertRaises(VCSError):
            self.tip.fill_archive(kind='wrong kind')

    def test_archive_empty_prefix(self):
        with self.assertRaises(VCSError):
            self.tip.fill_archive(prefix='')

    def test_archive_prefix_with_leading_slash(self):
        with self.assertRaises(VCSError):
            self.tip.fill_archive(prefix='/any')

# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s archive test' % alias).title().split())
    bases = (ArchivesTestCaseMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)

if __name__ == '__main__':
    unittest.main()

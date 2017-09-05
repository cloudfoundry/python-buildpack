from __future__ import with_statement

import datetime
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS
from vcs.nodes import FileNode
from vcs.utils.compat import unittest


class GetsliceTestCaseMixin(BackendTestMixin):

    @classmethod
    def _get_commits(cls):
        start_date = datetime.datetime(2010, 1, 1, 20)
        for x in xrange(5):
            yield {
                'message': 'Commit %d' % x,
                'author': 'Joe Doe <joe.doe@example.com>',
                'date': start_date + datetime.timedelta(hours=12 * x),
                'added': [
                    FileNode('file_%d.txt' % x, content='Foobar %d' % x),
                ],
            }

    def test__getslice__last_item_is_tip(self):
        self.assertEqual(list(self.repo[-1:])[0], self.repo.get_changeset())

    def test__getslice__respects_start_index(self):
        self.assertEqual(list(self.repo[2:]),
            [self.repo.get_changeset(rev) for rev in self.repo.revisions[2:]])

    def test__getslice__respects_negative_start_index(self):
        self.assertEqual(list(self.repo[-2:]),
            [self.repo.get_changeset(rev) for rev in self.repo.revisions[-2:]])

    def test__getslice__respects_end_index(self):
        self.assertEqual(list(self.repo[:2]),
            [self.repo.get_changeset(rev) for rev in self.repo.revisions[:2]])

    def test__getslice__respects_negative_end_index(self):
        self.assertEqual(list(self.repo[:-2]),
            [self.repo.get_changeset(rev) for rev in self.repo.revisions[:-2]])


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s getslice test' % alias).title().split())
    bases = (GetsliceTestCaseMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()

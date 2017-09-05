from __future__ import with_statement

import datetime
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS
from vcs.nodes import FileNode
from vcs.utils.compat import unittest


class GetitemTestCaseMixin(BackendTestMixin):

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

    def test__getitem__last_item_is_tip(self):
        self.assertEqual(self.repo[-1], self.repo.get_changeset())

    def test__getitem__returns_correct_items(self):
        changesets = [self.repo[x] for x in xrange(len(self.repo.revisions))]
        self.assertEqual(changesets, list(self.repo.get_changesets()))


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s getitem test' % alias).title().split())
    bases = (GetitemTestCaseMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()

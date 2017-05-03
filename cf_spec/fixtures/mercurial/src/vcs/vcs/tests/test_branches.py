from __future__ import with_statement

import datetime
import vcs
from vcs.utils.compat import unittest
from vcs.nodes import FileNode

from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS


class BranchesTestCaseMixin(BackendTestMixin):

    @classmethod
    def _get_commits(cls):
        commits = [
            {
                'message': 'Initial commit',
                'author': 'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [
                    FileNode('foobar', content='Foobar'),
                    FileNode('foobar2', content='Foobar II'),
                    FileNode('foo/bar/baz', content='baz here!'),
                ],
            },
            {
                'message': 'Changes...',
                'author': 'Jane Doe <jane.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 21),
                'added': [
                    FileNode('some/new.txt', content='news...'),
                ],
                'changed': [
                    FileNode('foobar', 'Foobar I'),
                ],
                'removed': [],
            },
        ]
        return commits

    def test_simple(self):
        tip = self.repo.get_changeset()
        self.assertEqual(tip.date, datetime.datetime(2010, 1, 1, 21))

    def test_new_branch(self):
        # This check must not be removed to ensure the 'branches' LazyProperty
        # gets hit *before* the new 'foobar' branch got created:
        self.assertFalse('foobar' in self.repo.branches)
        self.imc.add(vcs.nodes.FileNode('docs/index.txt',
            content='Documentation\n'))
        foobar_tip = self.imc.commit(
            message=u'New branch: foobar',
            author=u'joe',
            branch='foobar',
        )
        self.assertTrue('foobar' in self.repo.branches)
        self.assertEqual(foobar_tip.branch, 'foobar')

    def test_new_head(self):
        tip = self.repo.get_changeset()
        self.imc.add(vcs.nodes.FileNode('docs/index.txt',
            content='Documentation\n'))
        foobar_tip = self.imc.commit(
            message=u'New branch: foobar',
            author=u'joe',
            branch='foobar',
            parents=[tip],
        )
        self.imc.change(vcs.nodes.FileNode('docs/index.txt',
            content='Documentation\nand more...\n'))
        newtip = self.imc.commit(
            message=u'At default branch',
            author=u'joe',
            branch=foobar_tip.branch,
            parents=[foobar_tip],
        )

        newest_tip = self.imc.commit(
            message=u'Merged with %s' % foobar_tip.raw_id,
            author=u'joe',
            branch=self.backend_class.DEFAULT_BRANCH_NAME,
            parents=[newtip, foobar_tip],
        )

        self.assertEqual(newest_tip.branch,
            self.backend_class.DEFAULT_BRANCH_NAME)

    def test_branch_with_slash_in_name(self):
        self.imc.add(vcs.nodes.FileNode('extrafile', content='Some data\n'))
        self.imc.commit(u'Branch with a slash!', author=u'joe',
            branch='issue/123')
        self.assertTrue('issue/123' in self.repo.branches)

    def test_branch_with_slash_in_name_and_similar_without(self):
        self.imc.add(vcs.nodes.FileNode('extrafile', content='Some data\n'))
        self.imc.commit(u'Branch with a slash!', author=u'joe',
            branch='issue/123')
        self.imc.add(vcs.nodes.FileNode('extrafile II', content='Some data\n'))
        self.imc.commit(u'Branch without a slash...', author=u'joe',
            branch='123')
        self.assertIn('issue/123', self.repo.branches)
        self.assertIn('123', self.repo.branches)


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s branches test' % alias).title().split())
    bases = (BranchesTestCaseMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()

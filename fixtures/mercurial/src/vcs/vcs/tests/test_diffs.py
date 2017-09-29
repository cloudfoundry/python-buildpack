import datetime
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS
from vcs.nodes import FileNode
from vcs.utils.compat import unittest
from vcs.utils.diffs import get_gitdiff


class DiffsTestMixin(BackendTestMixin):

    @classmethod
    def _get_commits(cls):
        commits = [
            {
                'message': u'Initial commit',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [FileNode('file1', content='Foobar')],
            },
            {
                'message': u'Added a file2, change file1',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [FileNode('file2', content='Foobar')],
                'changed': [FileNode('file1', content='...')],
            },
            {
                'message': u'Remove file1',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'removed': [FileNode('file1')],
            },
        ]
        return commits

    def test_log_command(self):
        commits = [self.repo.get_changeset(r) for r in self.repo.revisions]
        commit1, commit2, commit3 = commits

        old = commit1.get_node('file1')
        new = commit2.get_node('file1')
        result = get_gitdiff(old, new).splitlines()
        # there are small differences between git and hg output so we explicitly
        # check only few things
        self.assertEqual(result[0], 'diff --git a/file1 b/file1')
        self.assertIn('-Foobar', result)
        self.assertIn('+...', result)


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s diff tests' % alias).title().split())
    bases = (DiffsTestMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()


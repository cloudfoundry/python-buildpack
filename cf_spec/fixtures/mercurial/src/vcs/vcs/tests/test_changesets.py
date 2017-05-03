from __future__ import with_statement

import datetime
import vcs
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS

from vcs.backends.base import BaseChangeset
from vcs.nodes import (
    FileNode, AddedFileNodesGenerator,
    ChangedFileNodesGenerator, RemovedFileNodesGenerator
)
from vcs.exceptions import (
    BranchDoesNotExistError, ChangesetDoesNotExistError,
    RepositoryError
)
from vcs.utils.compat import unittest


class TestBaseChangeset(unittest.TestCase):

    def test_as_dict(self):
        changeset = BaseChangeset()
        changeset.id = 'ID'
        changeset.raw_id = 'RAW_ID'
        changeset.short_id = 'SHORT_ID'
        changeset.revision = 1009
        changeset.date = datetime.datetime(2011, 1, 30, 1, 45)
        changeset.message = 'Message of a commit'
        changeset.author = 'Joe Doe <joe.doe@example.com>'
        changeset.added = [FileNode('foo/bar/baz'), FileNode('foobar')]
        changeset.changed = []
        changeset.removed = []
        self.assertEqual(changeset.as_dict(), {
            'id': 'ID',
            'raw_id': 'RAW_ID',
            'short_id': 'SHORT_ID',
            'revision': 1009,
            'date': datetime.datetime(2011, 1, 30, 1, 45),
            'message': 'Message of a commit',
            'author': {
                'name': 'Joe Doe',
                'email': 'joe.doe@example.com',
            },
            'added': ['foo/bar/baz', 'foobar'],
            'changed': [],
            'removed': [],
        })

class ChangesetsWithCommitsTestCaseixin(BackendTestMixin):
    recreate_repo_per_test = True

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

    def test_new_branch(self):
        self.imc.add(vcs.nodes.FileNode('docs/index.txt',
            content='Documentation\n'))
        foobar_tip = self.imc.commit(
            message=u'New branch: foobar',
            author=u'joe',
            branch='foobar',
        )
        self.assertTrue('foobar' in self.repo.branches)
        self.assertEqual(foobar_tip.branch, 'foobar')
        # 'foobar' should be the only branch that contains the new commit
        self.assertNotEqual(*self.repo.branches.values())

    def test_new_head_in_default_branch(self):
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

    def test_get_changesets_respects_branch_name(self):
        tip = self.repo.get_changeset()
        self.imc.add(vcs.nodes.FileNode('docs/index.txt',
            content='Documentation\n'))
        doc_changeset = self.imc.commit(
            message=u'New branch: docs',
            author=u'joe',
            branch='docs',
        )
        self.imc.add(vcs.nodes.FileNode('newfile', content=''))
        self.imc.commit(
            message=u'Back in default branch',
            author=u'joe',
            parents=[tip],
        )
        default_branch_changesets = self.repo.get_changesets(
            branch_name=self.repo.DEFAULT_BRANCH_NAME)
        self.assertNotIn(doc_changeset, default_branch_changesets)

    def test_get_changeset_by_branch(self):
        for branch, sha in self.repo.branches.iteritems():
            self.assertEqual(sha, self.repo.get_changeset(branch).raw_id)

    def test_get_changeset_by_tag(self):
        for tag, sha in self.repo.tags.iteritems():
            self.assertEqual(sha, self.repo.get_changeset(tag).raw_id)


class ChangesetsTestCaseMixin(BackendTestMixin):
    recreate_repo_per_test = False

    @classmethod
    def _get_commits(cls):
        start_date = datetime.datetime(2010, 1, 1, 20)
        for x in xrange(5):
            yield {
                'message': u'Commit %d' % x,
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': start_date + datetime.timedelta(hours=12 * x),
                'added': [
                    FileNode('file_%d.txt' % x, content='Foobar %d' % x),
                ],
            }

    def test_simple(self):
        tip = self.repo.get_changeset()
        self.assertEqual(tip.date, datetime.datetime(2010, 1, 3, 20))

    def test_get_changesets_is_ordered_by_date(self):
        changesets = list(self.repo.get_changesets())
        ordered_by_date = sorted(changesets,
            key=lambda cs: cs.date)
        self.assertItemsEqual(changesets, ordered_by_date)

    def test_get_changesets_respects_start(self):
        second_id = self.repo.revisions[1]
        changesets = list(self.repo.get_changesets(start=second_id))
        self.assertEqual(len(changesets), 4)

    def test_get_changesets_numerical_id_respects_start(self):
        second_id = 1
        changesets = list(self.repo.get_changesets(start=second_id))
        self.assertEqual(len(changesets), 4)

    def test_get_changesets_includes_start_changeset(self):
        second_id = self.repo.revisions[1]
        changesets = list(self.repo.get_changesets(start=second_id))
        self.assertEqual(changesets[0].raw_id, second_id)

    def test_get_changesets_respects_end(self):
        second_id = self.repo.revisions[1]
        changesets = list(self.repo.get_changesets(end=second_id))
        self.assertEqual(changesets[-1].raw_id, second_id)
        self.assertEqual(len(changesets), 2)

    def test_get_changesets_numerical_id_respects_end(self):
        second_id = 1
        changesets = list(self.repo.get_changesets(end=second_id))
        self.assertEqual(changesets.index(changesets[-1]), second_id)
        self.assertEqual(len(changesets), 2)

    def test_get_changesets_respects_both_start_and_end(self):
        second_id = self.repo.revisions[1]
        third_id = self.repo.revisions[2]
        changesets = list(self.repo.get_changesets(start=second_id,
            end=third_id))
        self.assertEqual(len(changesets), 2)

    def test_get_changesets_numerical_id_respects_both_start_and_end(self):
        changesets = list(self.repo.get_changesets(start=2, end=3))
        self.assertEqual(len(changesets), 2)

    def test_get_changesets_includes_end_changeset(self):
        second_id = self.repo.revisions[1]
        changesets = list(self.repo.get_changesets(end=second_id))
        self.assertEqual(changesets[-1].raw_id, second_id)

    def test_get_changesets_respects_start_date(self):
        start_date = datetime.datetime(2010, 2, 1)
        for cs in self.repo.get_changesets(start_date=start_date):
            self.assertGreaterEqual(cs.date, start_date)

    def test_get_changesets_respects_end_date(self):
        start_date = datetime.datetime(2010, 1, 1)
        end_date = datetime.datetime(2010, 2, 1)
        for cs in self.repo.get_changesets(start_date=start_date,
                                           end_date=end_date):
            self.assertGreaterEqual(cs.date, start_date)
            self.assertLessEqual(cs.date, end_date)

    def test_get_changesets_respects_start_date_and_end_date(self):
        end_date = datetime.datetime(2010, 2, 1)
        for cs in self.repo.get_changesets(end_date=end_date):
            self.assertLessEqual(cs.date, end_date)

    def test_get_changesets_respects_reverse(self):
        changesets_id_list = [cs.raw_id for cs in
            self.repo.get_changesets(reverse=True)]
        self.assertItemsEqual(changesets_id_list, reversed(self.repo.revisions))

    def test_get_filenodes_generator(self):
        tip = self.repo.get_changeset()
        filepaths = [node.path for node in tip.get_filenodes_generator()]
        self.assertItemsEqual(filepaths, ['file_%d.txt' % x for x in xrange(5)])

    def test_size(self):
        tip = self.repo.get_changeset()
        size = 5 * len('Foobar N') # Size of 5 files
        self.assertEqual(tip.size, size)

    def test_author(self):
        tip = self.repo.get_changeset()
        self.assertEqual(tip.author, u'Joe Doe <joe.doe@example.com>')

    def test_author_name(self):
        tip = self.repo.get_changeset()
        self.assertEqual(tip.author_name, u'Joe Doe')

    def test_author_email(self):
        tip = self.repo.get_changeset()
        self.assertEqual(tip.author_email, u'joe.doe@example.com')

    def test_get_changesets_raise_changesetdoesnotexist_for_wrong_start(self):
        with self.assertRaises(ChangesetDoesNotExistError):
            list(self.repo.get_changesets(start='foobar'))

    def test_get_changesets_raise_changesetdoesnotexist_for_wrong_end(self):
        with self.assertRaises(ChangesetDoesNotExistError):
            list(self.repo.get_changesets(end='foobar'))

    def test_get_changesets_raise_branchdoesnotexist_for_wrong_branch_name(self):
        with self.assertRaises(BranchDoesNotExistError):
            list(self.repo.get_changesets(branch_name='foobar'))

    def test_get_changesets_raise_repositoryerror_for_wrong_start_end(self):
        start = self.repo.revisions[-1]
        end = self.repo.revisions[0]
        with self.assertRaises(RepositoryError):
            list(self.repo.get_changesets(start=start, end=end))

    def test_get_changesets_numerical_id_reversed(self):
        with self.assertRaises(RepositoryError):
            [x for x in self.repo.get_changesets(start=3, end=2)]

    def test_get_changesets_numerical_id_respects_both_start_and_end_last(self):
        with self.assertRaises(RepositoryError):
            last = len(self.repo.revisions)
            list(self.repo.get_changesets(start=last-1, end=last-2))

    def test_get_changesets_numerical_id_last_zero_error(self):
        with self.assertRaises(RepositoryError):
            last = len(self.repo.revisions)
            list(self.repo.get_changesets(start=last-1, end=0))


class ChangesetsChangesTestCaseMixin(BackendTestMixin):
    recreate_repo_per_test = False

    @classmethod
    def _get_commits(cls):
        return [
            {
                'message': u'Initial',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [
                    FileNode('foo/bar', content='foo'),
                    FileNode('foobar', content='foo'),
                    FileNode('qwe', content='foo'),
                ],
            },
            {
                'message': u'Massive changes',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 22),
                'added': [FileNode('fallout', content='War never changes')],
                'changed': [
                    FileNode('foo/bar', content='baz'),
                    FileNode('foobar', content='baz'),
                ],
                'removed': [FileNode('qwe')],
            },
        ]

    def test_initial_commit(self):
        changeset = self.repo.get_changeset(0)
        self.assertItemsEqual(changeset.added, [
            changeset.get_node('foo/bar'),
            changeset.get_node('foobar'),
            changeset.get_node('qwe'),
        ])
        self.assertItemsEqual(changeset.changed, [])
        self.assertItemsEqual(changeset.removed, [])

    def test_head_added(self):
        changeset = self.repo.get_changeset()
        self.assertTrue(isinstance(changeset.added, AddedFileNodesGenerator))
        self.assertItemsEqual(changeset.added, [
            changeset.get_node('fallout'),
        ])
        self.assertTrue(isinstance(changeset.changed, ChangedFileNodesGenerator))
        self.assertItemsEqual(changeset.changed, [
            changeset.get_node('foo/bar'),
            changeset.get_node('foobar'),
        ])
        self.assertTrue(isinstance(changeset.removed, RemovedFileNodesGenerator))
        self.assertEqual(len(changeset.removed), 1)
        self.assertEqual(list(changeset.removed)[0].path, 'qwe')


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    # tests with additional commits
    cls_name = ''.join(('%s changesets with commits test' % alias).title().split())
    bases = (ChangesetsWithCommitsTestCaseixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)

    # tests without additional commits
    cls_name = ''.join(('%s changesets test' % alias).title().split())
    bases = (ChangesetsTestCaseMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)

    # tests changes
    cls_name = ''.join(('%s changesets changes test' % alias).title().split())
    bases = (ChangesetsChangesTestCaseMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()

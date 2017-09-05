"""
Tests so called "in memory changesets" commit API of vcs.
"""
from __future__ import with_statement

import time
import datetime

import vcs
from vcs.tests.conf import SCM_TESTS, get_new_dir
from vcs.exceptions import EmptyRepositoryError
from vcs.exceptions import NodeAlreadyAddedError
from vcs.exceptions import NodeAlreadyExistsError
from vcs.exceptions import NodeAlreadyRemovedError
from vcs.exceptions import NodeAlreadyChangedError
from vcs.exceptions import NodeDoesNotExistError
from vcs.exceptions import NodeNotChangedError
from vcs.nodes import DirNode
from vcs.nodes import FileNode
from vcs.utils.compat import unittest


class InMemoryChangesetTestMixin(object):
    """
    This is a backend independent test case class which should be created
    with ``type`` method.

    It is required to set following attributes at subclass:

    - ``backend_alias``: alias of used backend (see ``vcs.BACKENDS``)
    - ``repo_path``: path to the repository which would be created for set of
      tests
    """

    def get_backend(self):
        return vcs.get_backend(self.backend_alias)

    def setUp(self):
        Backend = self.get_backend()
        self.repo_path = get_new_dir(str(time.time()))
        self.repo = Backend(self.repo_path, create=True)
        self.imc = self.repo.in_memory_changeset
        self.nodes = [
            FileNode('foobar', content='Foo & bar'),
            FileNode('foobar2', content='Foo & bar, doubled!'),
            FileNode('foo bar with spaces', content=''),
            FileNode('foo/bar/baz', content='Inside'),
            FileNode('foo/bar/file.bin', content='\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00;\x00\x03\x00\xfe\xff\t\x00\x06\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x1a\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x18\x00\x00\x00\x01\x00\x00\x00\xfe\xff\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff'),
        ]

    def test_add(self):
        rev_count = len(self.repo.revisions)
        to_add = [FileNode(node.path, content=node.content)
            for node in self.nodes]
        for node in to_add:
            self.imc.add(node)
        message = u'Added: %s' % ', '.join((node.path for node in self.nodes))
        author = unicode(self.__class__)
        changeset = self.imc.commit(message=message, author=author)

        newtip = self.repo.get_changeset()
        self.assertEqual(changeset, newtip)
        self.assertEqual(rev_count + 1, len(self.repo.revisions))
        self.assertEqual(newtip.message, message)
        self.assertEqual(newtip.author, author)
        self.assertTrue(not any((self.imc.added, self.imc.changed,
            self.imc.removed)))
        for node in to_add:
            self.assertEqual(newtip.get_node(node.path).content, node.content)

    def test_add_in_bulk(self):
        rev_count = len(self.repo.revisions)
        to_add = [FileNode(node.path, content=node.content)
            for node in self.nodes]
        self.imc.add(*to_add)
        message = u'Added: %s' % ', '.join((node.path for node in self.nodes))
        author = unicode(self.__class__)
        changeset = self.imc.commit(message=message, author=author)

        newtip = self.repo.get_changeset()
        self.assertEqual(changeset, newtip)
        self.assertEqual(rev_count + 1, len(self.repo.revisions))
        self.assertEqual(newtip.message, message)
        self.assertEqual(newtip.author, author)
        self.assertTrue(not any((self.imc.added, self.imc.changed,
            self.imc.removed)))
        for node in to_add:
            self.assertEqual(newtip.get_node(node.path).content, node.content)

    def test_add_actually_adds_all_nodes_at_second_commit_too(self):
        self.imc.add(FileNode('foo/bar/image.png', content='\0'))
        self.imc.add(FileNode('foo/README.txt', content='readme!'))
        changeset = self.imc.commit(u'Initial', u'joe.doe@example.com')
        self.assertTrue(isinstance(changeset.get_node('foo'), DirNode))
        self.assertTrue(isinstance(changeset.get_node('foo/bar'), DirNode))
        self.assertEqual(changeset.get_node('foo/bar/image.png').content, '\0')
        self.assertEqual(changeset.get_node('foo/README.txt').content, 'readme!')

        # commit some more files again
        to_add = [
            FileNode('foo/bar/foobaz/bar', content='foo'),
            FileNode('foo/bar/another/bar', content='foo'),
            FileNode('foo/baz.txt', content='foo'),
            FileNode('foobar/foobaz/file', content='foo'),
            FileNode('foobar/barbaz', content='foo'),
        ]
        self.imc.add(*to_add)
        changeset = self.imc.commit(u'Another', u'joe.doe@example.com')
        self.assertEqual(changeset.get_node('foo/bar/foobaz/bar').content, 'foo')
        self.assertEqual(changeset.get_node('foo/bar/another/bar').content, 'foo')
        self.assertEqual(changeset.get_node('foo/baz.txt').content, 'foo')
        self.assertEqual(changeset.get_node('foobar/foobaz/file').content, 'foo')
        self.assertEqual(changeset.get_node('foobar/barbaz').content, 'foo')

    def test_add_raise_already_added(self):
        node = FileNode('foobar', content='baz')
        self.imc.add(node)
        self.assertRaises(NodeAlreadyAddedError, self.imc.add, node)

    def test_check_integrity_raise_already_exist(self):
        node = FileNode('foobar', content='baz')
        self.imc.add(node)
        self.imc.commit(message=u'Added foobar', author=unicode(self))
        self.imc.add(node)
        self.assertRaises(NodeAlreadyExistsError, self.imc.commit,
            message='new message',
            author=str(self))

    def test_change(self):
        self.imc.add(FileNode('foo/bar/baz', content='foo'))
        self.imc.add(FileNode('foo/fbar', content='foobar'))
        tip = self.imc.commit(u'Initial', u'joe.doe@example.com')

        # Change node's content
        node = FileNode('foo/bar/baz', content='My **changed** content')
        self.imc.change(node)
        self.imc.commit(u'Changed %s' % node.path, u'joe.doe@example.com')

        newtip = self.repo.get_changeset()
        self.assertNotEqual(tip, newtip)
        self.assertNotEqual(tip.id, newtip.id)
        self.assertEqual(newtip.get_node('foo/bar/baz').content,
            'My **changed** content')

    def test_change_raise_empty_repository(self):
        node = FileNode('foobar')
        self.assertRaises(EmptyRepositoryError, self.imc.change, node)

    def test_check_integrity_change_raise_node_does_not_exist(self):
        node = FileNode('foobar', content='baz')
        self.imc.add(node)
        self.imc.commit(message=u'Added foobar', author=unicode(self))
        node = FileNode('not-foobar', content='')
        self.imc.change(node)
        self.assertRaises(NodeDoesNotExistError, self.imc.commit,
            message='Changed not existing node',
            author=str(self))

    def test_change_raise_node_already_changed(self):
        node = FileNode('foobar', content='baz')
        self.imc.add(node)
        self.imc.commit(message=u'Added foobar', author=unicode(self))
        node = FileNode('foobar', content='more baz')
        self.imc.change(node)
        self.assertRaises(NodeAlreadyChangedError, self.imc.change, node)

    def test_check_integrity_change_raise_node_not_changed(self):
        self.test_add()  # Performs first commit

        node = FileNode(self.nodes[0].path, content=self.nodes[0].content)
        self.imc.change(node)
        self.assertRaises(NodeNotChangedError, self.imc.commit,
            message=u'Trying to mark node as changed without touching it',
            author=unicode(self))

    def test_change_raise_node_already_removed(self):
        node = FileNode('foobar', content='baz')
        self.imc.add(node)
        self.imc.commit(message=u'Added foobar', author=unicode(self))
        self.imc.remove(FileNode('foobar'))
        self.assertRaises(NodeAlreadyRemovedError, self.imc.change, node)

    def test_remove(self):
        self.test_add()  # Performs first commit

        tip = self.repo.get_changeset()
        node = self.nodes[0]
        self.assertEqual(node.content, tip.get_node(node.path).content)
        self.imc.remove(node)
        self.imc.commit(message=u'Removed %s' % node.path, author=unicode(self))

        newtip = self.repo.get_changeset()
        self.assertNotEqual(tip, newtip)
        self.assertNotEqual(tip.id, newtip.id)
        self.assertRaises(NodeDoesNotExistError, newtip.get_node, node.path)

    def test_remove_last_file_from_directory(self):
        node = FileNode('omg/qwe/foo/bar', content='foobar')
        self.imc.add(node)
        self.imc.commit(u'added', u'joe doe')

        self.imc.remove(node)
        tip = self.imc.commit(u'removed', u'joe doe')
        self.assertRaises(NodeDoesNotExistError, tip.get_node, 'omg/qwe/foo/bar')

    def test_remove_raise_node_does_not_exist(self):
        self.imc.remove(self.nodes[0])
        self.assertRaises(NodeDoesNotExistError, self.imc.commit,
            message='Trying to remove node at empty repository',
            author=str(self))

    def test_check_integrity_remove_raise_node_does_not_exist(self):
        self.test_add()  # Performs first commit

        node = FileNode('no-such-file')
        self.imc.remove(node)
        self.assertRaises(NodeDoesNotExistError, self.imc.commit,
            message=u'Trying to remove not existing node',
            author=unicode(self))

    def test_remove_raise_node_already_removed(self):
        self.test_add() # Performs first commit

        node = FileNode(self.nodes[0].path)
        self.imc.remove(node)
        self.assertRaises(NodeAlreadyRemovedError, self.imc.remove, node)

    def test_remove_raise_node_already_changed(self):
        self.test_add()  # Performs first commit

        node = FileNode(self.nodes[0].path, content='Bending time')
        self.imc.change(node)
        self.assertRaises(NodeAlreadyChangedError, self.imc.remove, node)

    def test_reset(self):
        self.imc.add(FileNode('foo', content='bar'))
        #self.imc.change(FileNode('baz', content='new'))
        #self.imc.remove(FileNode('qwe'))
        self.imc.reset()
        self.assertTrue(not any((self.imc.added, self.imc.changed,
            self.imc.removed)))

    def test_multiple_commits(self):
        N = 3  # number of commits to perform
        last = None
        for x in xrange(N):
            fname = 'file%s' % str(x).rjust(5, '0')
            content = 'foobar\n' * x
            node = FileNode(fname, content=content)
            self.imc.add(node)
            commit = self.imc.commit(u"Commit no. %s" % (x + 1), author=u'vcs')
            self.assertTrue(last != commit)
            last = commit

        # Check commit number for same repo
        self.assertEqual(len(self.repo.revisions), N)

        # Check commit number for recreated repo
        backend = self.get_backend()
        repo = backend(self.repo_path)
        self.assertEqual(len(repo.revisions), N)

    def test_date_attr(self):
        node = FileNode('foobar.txt', content='Foobared!')
        self.imc.add(node)
        date = datetime.datetime(1985, 1, 30, 1, 45)
        commit = self.imc.commit(u"Committed at time when I was born ;-)",
            author=u'lb', date=date)

        self.assertEqual(commit.date, date)


class BackendBaseTestCase(unittest.TestCase):
    """
    Base test class for tests which requires repository.
    """
    backend_alias = 'hg'
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
    ]

    def get_backend(self):
        return vcs.get_backend(self.backend_alias)

    def get_commits(self):
        """
        Returns list of commits which builds repository for each tests.
        """
        if hasattr(self, 'commits'):
            return self.commits

    def get_new_repo_path(self):
        """
        Returns newly created repository's directory.
        """
        backend = self.get_backend()
        key = '%s-%s' % (backend.alias, str(time.time()))
        repo_path = get_new_dir(key)
        return repo_path

    def setUp(self):
        Backend = self.get_backend()
        self.backend_class = Backend
        self.repo_path = self.get_new_repo_path()
        self.repo = Backend(self.repo_path, create=True)
        self.imc = self.repo.in_memory_changeset

        for commit in self.get_commits():
            for node in commit.get('added', []):
                self.imc.add(FileNode(node.path, content=node.content))
            for node in commit.get('changed', []):
                self.imc.change(FileNode(node.path, content=node.content))
            for node in commit.get('removed', []):
                self.imc.remove(FileNode(node.path))
            self.imc.commit(message=unicode(commit['message']),
                            author=unicode(commit['author']),
                date=commit['date'])

        self.tip = self.repo.get_changeset()


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s in memory changeset test' % alias).title().split())
    bases = (InMemoryChangesetTestMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()

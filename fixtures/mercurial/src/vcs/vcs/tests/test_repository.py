from __future__ import with_statement
import datetime
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS
from vcs.tests.conf import TEST_USER_CONFIG_FILE
from vcs.nodes import FileNode
from vcs.utils.compat import unittest
from vcs.exceptions import ChangesetDoesNotExistError


class RepositoryBaseTest(BackendTestMixin):
    recreate_repo_per_test = False

    @classmethod
    def _get_commits(cls):
        return super(RepositoryBaseTest, cls)._get_commits()[:1]

    def test_get_config_value(self):
        self.assertEqual(self.repo.get_config_value('universal', 'foo',
            TEST_USER_CONFIG_FILE), 'bar')

    def test_get_config_value_defaults_to_None(self):
        self.assertEqual(self.repo.get_config_value('universal', 'nonexist',
            TEST_USER_CONFIG_FILE), None)

    def test_get_user_name(self):
        self.assertEqual(self.repo.get_user_name(TEST_USER_CONFIG_FILE),
            'Foo Bar')

    def test_get_user_email(self):
        self.assertEqual(self.repo.get_user_email(TEST_USER_CONFIG_FILE),
            'foo.bar@example.com')

    def test_repo_equality(self):
        self.assertTrue(self.repo == self.repo)

    def test_repo_equality_broken_object(self):
        import copy
        _repo = copy.copy(self.repo)
        delattr(_repo, 'path')
        self.assertTrue(self.repo != _repo)

    def test_repo_equality_other_object(self):
        class dummy(object):
            path = self.repo.path
        self.assertTrue(self.repo != dummy())


class RepositoryGetDiffTest(BackendTestMixin):

    @classmethod
    def _get_commits(cls):
        commits = [
            {
                'message': 'Initial commit',
                'author': 'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [
                    FileNode('foobar', content='foobar'),
                    FileNode('foobar2', content='foobar2'),
                ],
            },
            {
                'message': 'Changed foobar, added foobar3',
                'author': 'Jane Doe <jane.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 21),
                'added': [
                    FileNode('foobar3', content='foobar3'),
                ],
                'changed': [
                    FileNode('foobar', 'FOOBAR'),
                ],
            },
            {
                'message': 'Removed foobar, changed foobar3',
                'author': 'Jane Doe <jane.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 22),
                'changed': [
                    FileNode('foobar3', content='FOOBAR\nFOOBAR\nFOOBAR\n'),
                ],
                'removed': [FileNode('foobar')],
            },
        ]
        return commits

    def test_raise_for_wrong(self):
        with self.assertRaises(ChangesetDoesNotExistError):
            self.repo.get_diff('a' * 40, 'b' * 40)


class GitRepositoryGetDiffTest(RepositoryGetDiffTest, unittest.TestCase):
    backend_alias = 'git'

    def test_initial_commit_diff(self):
        initial_rev = self.repo.revisions[0]
        self.assertEqual(self.repo.get_diff(self.repo.EMPTY_CHANGESET, initial_rev), '''diff --git a/foobar b/foobar
new file mode 100644
index 0000000000000000000000000000000000000000..f6ea0495187600e7b2288c8ac19c5886383a4632
--- /dev/null
+++ b/foobar
@@ -0,0 +1 @@
+foobar
\ No newline at end of file
diff --git a/foobar2 b/foobar2
new file mode 100644
index 0000000000000000000000000000000000000000..e8c9d6b98e3dce993a464935e1a53f50b56a3783
--- /dev/null
+++ b/foobar2
@@ -0,0 +1 @@
+foobar2
\ No newline at end of file
''')

    def test_second_changeset_diff(self):
        revs = self.repo.revisions
        self.assertEqual(self.repo.get_diff(revs[0], revs[1]), '''diff --git a/foobar b/foobar
index f6ea0495187600e7b2288c8ac19c5886383a4632..389865bb681b358c9b102d79abd8d5f941e96551 100644
--- a/foobar
+++ b/foobar
@@ -1 +1 @@
-foobar
\ No newline at end of file
+FOOBAR
\ No newline at end of file
diff --git a/foobar3 b/foobar3
new file mode 100644
index 0000000000000000000000000000000000000000..c11c37d41d33fb47741cff93fa5f9d798c1535b0
--- /dev/null
+++ b/foobar3
@@ -0,0 +1 @@
+foobar3
\ No newline at end of file
''')

    def test_third_changeset_diff(self):
        revs = self.repo.revisions
        self.assertEqual(self.repo.get_diff(revs[1], revs[2]), '''diff --git a/foobar b/foobar
deleted file mode 100644
index 389865bb681b358c9b102d79abd8d5f941e96551..0000000000000000000000000000000000000000
--- a/foobar
+++ /dev/null
@@ -1 +0,0 @@
-FOOBAR
\ No newline at end of file
diff --git a/foobar3 b/foobar3
index c11c37d41d33fb47741cff93fa5f9d798c1535b0..f9324477362684ff692aaf5b9a81e01b9e9a671c 100644
--- a/foobar3
+++ b/foobar3
@@ -1 +1,3 @@
-foobar3
\ No newline at end of file
+FOOBAR
+FOOBAR
+FOOBAR
''')


class HgRepositoryGetDiffTest(RepositoryGetDiffTest, unittest.TestCase):
    backend_alias = 'hg'

    def test_initial_commit_diff(self):
        initial_rev = self.repo.revisions[0]
        self.assertEqual(self.repo.get_diff(self.repo.EMPTY_CHANGESET, initial_rev), '''diff --git a/foobar b/foobar
new file mode 100755
--- /dev/null
+++ b/foobar
@@ -0,0 +1,1 @@
+foobar
\ No newline at end of file
diff --git a/foobar2 b/foobar2
new file mode 100755
--- /dev/null
+++ b/foobar2
@@ -0,0 +1,1 @@
+foobar2
\ No newline at end of file
''')

    def test_second_changeset_diff(self):
        revs = self.repo.revisions
        self.assertEqual(self.repo.get_diff(revs[0], revs[1]), '''diff --git a/foobar b/foobar
--- a/foobar
+++ b/foobar
@@ -1,1 +1,1 @@
-foobar
\ No newline at end of file
+FOOBAR
\ No newline at end of file
diff --git a/foobar3 b/foobar3
new file mode 100755
--- /dev/null
+++ b/foobar3
@@ -0,0 +1,1 @@
+foobar3
\ No newline at end of file
''')

    def test_third_changeset_diff(self):
        revs = self.repo.revisions
        self.assertEqual(self.repo.get_diff(revs[1], revs[2]), '''diff --git a/foobar b/foobar
deleted file mode 100755
--- a/foobar
+++ /dev/null
@@ -1,1 +0,0 @@
-FOOBAR
\ No newline at end of file
diff --git a/foobar3 b/foobar3
--- a/foobar3
+++ b/foobar3
@@ -1,1 +1,3 @@
-foobar3
\ No newline at end of file
+FOOBAR
+FOOBAR
+FOOBAR
''')


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = alias.capitalize() + RepositoryBaseTest.__name__
    bases = (RepositoryBaseTest, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)

if __name__ == '__main__':
    unittest.main()

from __future__ import with_statement

import os
import shutil

from vcs import VCSError, get_repo, get_backend
from vcs.backends.hg import MercurialRepository
from vcs.utils.compat import unittest
from vcs.tests.conf import TEST_HG_REPO, TEST_GIT_REPO, TEST_TMP_PATH



class VCSTest(unittest.TestCase):
    """
    Tests for main module's methods.
    """

    def test_get_backend(self):
        hg = get_backend('hg')
        self.assertEqual(hg, MercurialRepository)

    def test_alias_detect_hg(self):
        alias = 'hg'
        path = TEST_HG_REPO
        backend = get_backend(alias)
        repo = backend(path)
        self.assertEqual('hg',repo.alias)

    def test_alias_detect_git(self):
        alias = 'git'
        path = TEST_GIT_REPO
        backend = get_backend(alias)
        repo = backend(path)
        self.assertEqual('git',repo.alias)

    def test_wrong_alias(self):
        alias = 'wrong_alias'
        self.assertRaises(VCSError, get_backend, alias)

    def test_get_repo(self):
        alias = 'hg'
        path = TEST_HG_REPO
        backend = get_backend(alias)
        repo = backend(path)

        self.assertEqual(repo.__class__, get_repo(path, alias).__class__)
        self.assertEqual(repo.path, get_repo(path, alias).path)

    def test_get_repo_autoalias_hg(self):
        alias = 'hg'
        path = TEST_HG_REPO
        backend = get_backend(alias)
        repo = backend(path)

        self.assertEqual(repo.__class__, get_repo(path).__class__)
        self.assertEqual(repo.path, get_repo(path).path)

    def test_get_repo_autoalias_git(self):
        alias = 'git'
        path = TEST_GIT_REPO
        backend = get_backend(alias)
        repo = backend(path)

        self.assertEqual(repo.__class__, get_repo(path).__class__)
        self.assertEqual(repo.path, get_repo(path).path)


    def test_get_repo_err(self):
        blank_repo_path = os.path.join(TEST_TMP_PATH, 'blank-error-repo')
        if os.path.isdir(blank_repo_path):
            shutil.rmtree(blank_repo_path)

        os.mkdir(blank_repo_path)
        self.assertRaises(VCSError, get_repo, blank_repo_path)
        self.assertRaises(VCSError, get_repo, blank_repo_path + 'non_existing')

    def test_get_repo_multialias(self):
        multialias_repo_path = os.path.join(TEST_TMP_PATH, 'hg-git-repo')
        if os.path.isdir(multialias_repo_path):
            shutil.rmtree(multialias_repo_path)

        os.mkdir(multialias_repo_path)

        os.mkdir(os.path.join(multialias_repo_path, '.git'))
        os.mkdir(os.path.join(multialias_repo_path, '.hg'))
        self.assertRaises(VCSError, get_repo, multialias_repo_path)

"""
Unit tests configuration module for vcs.
"""
import os
import time
import hashlib
import tempfile
import datetime
import shutil
from utils import get_normalized_path
from os.path import join as jn

__all__ = (
    'TEST_HG_REPO', 'TEST_GIT_REPO', 'HG_REMOTE_REPO', 'GIT_REMOTE_REPO',
    'SCM_TESTS',
)

SCM_TESTS = ['hg', 'git']
uniq_suffix = str(int(time.mktime(datetime.datetime.now().timetuple())))

THIS = os.path.abspath(os.path.dirname(__file__))

GIT_REMOTE_REPO = 'git://github.com/codeinn/vcs.git'

TEST_TMP_PATH = os.environ.get('VCS_TEST_ROOT', '/tmp')
TEST_GIT_REPO = os.environ.get('VCS_TEST_GIT_REPO',
                              jn(TEST_TMP_PATH, 'vcs-git'))
TEST_GIT_REPO_CLONE = os.environ.get('VCS_TEST_GIT_REPO_CLONE',
                            jn(TEST_TMP_PATH, 'vcsgitclone%s' % uniq_suffix))
TEST_GIT_REPO_PULL = os.environ.get('VCS_TEST_GIT_REPO_PULL',
                            jn(TEST_TMP_PATH, 'vcsgitpull%s' % uniq_suffix))

HG_REMOTE_REPO = 'http://bitbucket.org/marcinkuzminski/vcs'
TEST_HG_REPO = os.environ.get('VCS_TEST_HG_REPO',
                              jn(TEST_TMP_PATH, 'vcs-hg'))
TEST_HG_REPO_CLONE = os.environ.get('VCS_TEST_HG_REPO_CLONE',
                              jn(TEST_TMP_PATH, 'vcshgclone%s' % uniq_suffix))
TEST_HG_REPO_PULL = os.environ.get('VCS_TEST_HG_REPO_PULL',
                              jn(TEST_TMP_PATH, 'vcshgpull%s' % uniq_suffix))

TEST_DIR = os.environ.get('VCS_TEST_ROOT', tempfile.gettempdir())
TEST_REPO_PREFIX = 'vcs-test'


def get_new_dir(title):
    """
    Returns always new directory path.
    """
    name = TEST_REPO_PREFIX
    if title:
        name = '-'.join((name, title))
    hex = hashlib.sha1(str(time.time())).hexdigest()
    name = '-'.join((name, hex))
    path = os.path.join(TEST_DIR, name)
    return get_normalized_path(path)


PACKAGE_DIR = os.path.abspath(os.path.join(
    os.path.dirname(__file__), '..'))
_dest = jn(TEST_TMP_PATH, 'aconfig')
shutil.copy(jn(THIS, 'aconfig'), _dest)
TEST_USER_CONFIG_FILE = _dest

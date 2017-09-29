import os
import tempfile
from vcs.utils import aslist
from vcs.utils.paths import get_user_home

abspath = lambda * p: os.path.abspath(os.path.join(*p))

VCSRC_PATH = os.environ.get('VCSRC_PATH')

if not VCSRC_PATH:
    HOME_ = get_user_home()
    if not HOME_:
        HOME_ = tempfile.gettempdir()

VCSRC_PATH = VCSRC_PATH or abspath(HOME_, '.vcsrc')
if os.path.isdir(VCSRC_PATH):
    VCSRC_PATH = os.path.join(VCSRC_PATH, '__init__.py')

# list of default encoding used in safe_unicode/safe_str methods
DEFAULT_ENCODINGS = aslist('utf8')

# path to git executable runned by run_git_command function
GIT_EXECUTABLE_PATH = 'git'
# can be also --branches --tags
GIT_REV_FILTER = '--all'

BACKENDS = {
    'hg': 'vcs.backends.hg.MercurialRepository',
    'git': 'vcs.backends.git.GitRepository',
}

ARCHIVE_SPECS = {
    'tar': ('application/x-tar', '.tar'),
    'tbz2': ('application/x-bzip2', '.tar.bz2'),
    'tgz': ('application/x-gzip', '.tar.gz'),
    'zip': ('application/zip', '.zip'),
}

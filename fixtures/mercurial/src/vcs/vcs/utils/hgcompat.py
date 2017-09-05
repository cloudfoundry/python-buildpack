"""
Mercurial libs compatibility
"""

from mercurial import archival, merge as hg_merge, patch, ui
from mercurial.commands import clone, nullid, pull
from mercurial.context import memctx, memfilectx
from mercurial.error import RepoError, RepoLookupError, Abort
from mercurial.hgweb.common import get_contact
from mercurial.localrepo import localrepository
from mercurial.match import match
from mercurial.mdiff import diffopts
from mercurial.node import hex
from mercurial.encoding import tolocal
from mercurial import discovery
from mercurial import localrepo
from mercurial import scmutil
from mercurial.discovery import findcommonoutgoing

from mercurial.util import url as hg_url

# those authnadlers are patched for python 2.6.5 bug an
# infinit looping when given invalid resources
from mercurial.url import httpbasicauthhandler, httpdigestauthhandler

# -*- coding: utf-8 -*-
"""
    vcs.backends.git.repository
    ~~~~~~~~~~~~~~~~~~~~~~~~~~~

    Git repository implementation.

    :created_on: Apr 8, 2010
    :copyright: (c) 2010-2011 by Marcin Kuzminski, Lukasz Balcerzak.
"""

import os
import re
import time
import urllib
import urllib2
import logging
import posixpath
import string

from dulwich.objects import Tag
from dulwich.repo import Repo, NotGitRepository

from vcs import subprocessio
from vcs.backends.base import BaseRepository, CollectionGenerator
from vcs.conf import settings

from vcs.exceptions import (
    BranchDoesNotExistError, ChangesetDoesNotExistError, EmptyRepositoryError,
    RepositoryError, TagAlreadyExistError, TagDoesNotExistError
)
from vcs.utils import safe_unicode, makedate, date_fromtimestamp
from vcs.utils.lazy import LazyProperty
from vcs.utils.ordered_dict import OrderedDict
from vcs.utils.paths import abspath, get_user_home

from vcs.utils.hgcompat import (
    hg_url, httpbasicauthhandler, httpdigestauthhandler
)

from .changeset import GitChangeset
from .config import ConfigFile
from .inmemory import GitInMemoryChangeset
from .workdir import GitWorkdir

SHA_PATTERN = re.compile(r'^[[0-9a-fA-F]{12}|[0-9a-fA-F]{40}]$')


class GitRepository(BaseRepository):
    """
    Git repository backend.
    """
    DEFAULT_BRANCH_NAME = 'master'
    scm = 'git'

    def __init__(self, repo_path, create=False, src_url=None,
                 update_after_clone=False, bare=False):

        self.path = abspath(repo_path)
        repo = self._get_repo(create, src_url, update_after_clone, bare)
        self.bare = repo.bare

    @property
    def _config_files(self):
        return [
            self.bare and abspath(self.path, 'config')
                      or abspath(self.path, '.git', 'config'),
             abspath(get_user_home(), '.gitconfig'),
         ]

    @property
    def _repo(self):
        return Repo(self.path)

    @property
    def head(self):
        try:
            return self._repo.head()
        except KeyError:
            return None

    @LazyProperty
    def revisions(self):
        """
        Returns list of revisions' ids, in ascending order.  Being lazy
        attribute allows external tools to inject shas from cache.
        """
        return self._get_all_revisions()

    @classmethod
    def _run_git_command(cls, cmd, **opts):
        """
        Runs given ``cmd`` as git command and returns tuple
        (stdout, stderr).

        :param cmd: git command to be executed
        :param opts: env options to pass into Subprocess command
        """

        if '_bare' in opts:
            _copts = []
            del opts['_bare']
        else:
            _copts = ['-c', 'core.quotepath=false', ]
        safe_call = False
        if '_safe' in opts:
            #no exc on failure
            del opts['_safe']
            safe_call = True

        _str_cmd = False
        if isinstance(cmd, basestring):
            cmd = [cmd]
            _str_cmd = True

        gitenv = os.environ
        # need to clean fix GIT_DIR !
        if 'GIT_DIR' in gitenv:
            del gitenv['GIT_DIR']
        gitenv['GIT_CONFIG_NOGLOBAL'] = '1'

        _git_path = settings.GIT_EXECUTABLE_PATH
        cmd = [_git_path] + _copts + cmd
        if _str_cmd:
            cmd = ' '.join(cmd)
        try:
            _opts = dict(
                env=gitenv,
                shell=False,
            )
            _opts.update(opts)
            p = subprocessio.SubprocessIOChunker(cmd, **_opts)
        except (EnvironmentError, OSError), err:
            tb_err = ("Couldn't run git command (%s).\n"
                      "Original error was:%s\n" % (cmd, err))
            log.error(tb_err)
            if safe_call:
                return '', err
            else:
                raise RepositoryError(tb_err)

        return ''.join(p.output), ''.join(p.error)

    def run_git_command(self, cmd):
        opts = {}
        if os.path.isdir(self.path):
            opts['cwd'] = self.path
        return self._run_git_command(cmd, **opts)

    @classmethod
    def _check_url(cls, url):
        """
        Functon will check given url and try to verify if it's a valid
        link. Sometimes it may happened that mercurial will issue basic
        auth request that can cause whole API to hang when used from python
        or other external calls.

        On failures it'll raise urllib2.HTTPError
        """

        # check first if it's not an local url
        if os.path.isdir(url) or url.startswith('file:'):
            return True

        if('+' in url[:url.find('://')]):
            url = url[url.find('+') + 1:]

        handlers = []
        test_uri, authinfo = hg_url(url).authinfo()
        if not test_uri.endswith('info/refs'):
            test_uri = test_uri.rstrip('/') + '/info/refs'
        if authinfo:
            #create a password manager
            passmgr = urllib2.HTTPPasswordMgrWithDefaultRealm()
            passmgr.add_password(*authinfo)

            handlers.extend((httpbasicauthhandler(passmgr),
                             httpdigestauthhandler(passmgr)))

        o = urllib2.build_opener(*handlers)
        o.addheaders = [('User-Agent', 'git/1.7.8.0')]  # fake some git

        q = {"service": 'git-upload-pack'}
        qs = '?%s' % urllib.urlencode(q)
        cu = "%s%s" % (test_uri, qs)
        req = urllib2.Request(cu, None, {})

        try:
            resp = o.open(req)
            return resp.code == 200
        except Exception, e:
            # means it cannot be cloned
            raise urllib2.URLError("[%s] %s" % (url, e))

    def _get_repo(self, create, src_url=None, update_after_clone=False,
                  bare=False):
        if create and os.path.exists(self.path):
            raise RepositoryError("Location already exist")
        if src_url and not create:
            raise RepositoryError("Create should be set to True if src_url is "
                                  "given (clone operation creates repository)")
        try:
            if create and src_url:
                GitRepository._check_url(src_url)
                self.clone(src_url, update_after_clone, bare)
                return Repo(self.path)
            elif create:
                os.mkdir(self.path)
                if bare:
                    return Repo.init_bare(self.path)
                else:
                    return Repo.init(self.path)
            else:
                return self._repo
        except (NotGitRepository, OSError), err:
            raise RepositoryError(err)

    def _get_all_revisions(self):
        # we must check if this repo is not empty, since later command
        # fails if it is. And it's cheaper to ask than throw the subprocess
        # errors
        try:
            self._repo.head()
        except KeyError:
            return []

        rev_filter = _git_path = settings.GIT_REV_FILTER
        cmd = 'rev-list %s --reverse --date-order' % (rev_filter)
        try:
            so, se = self.run_git_command(cmd)
        except RepositoryError:
            # Can be raised for empty repositories
            return []
        return so.splitlines()

    def _get_all_revisions2(self):
        #alternate implementation using dulwich
        includes = [x[1][0] for x in self._parsed_refs.iteritems()
                    if x[1][1] != 'T']
        return [c.commit.id for c in self._repo.get_walker(include=includes)]

    def _get_revision(self, revision):
        """
        For git backend we always return integer here. This way we ensure
        that changset's revision attribute would become integer.
        """

        is_null = lambda o: len(o) == revision.count('0')

        try:
            self.revisions[0]
        except (KeyError, IndexError):
            raise EmptyRepositoryError("There are no changesets yet")

        if revision in (None, '', 'tip', 'HEAD', 'head', -1):
            return self.revisions[-1]

        is_bstr = isinstance(revision, (str, unicode))
        if ((is_bstr and revision.isdigit() and len(revision) < 12)
            or isinstance(revision, int) or is_null(revision)):
            try:
                revision = self.revisions[int(revision)]
            except Exception:
                raise ChangesetDoesNotExistError("Revision %s does not exist "
                    "for this repository" % (revision))

        elif is_bstr:
            # get by branch/tag name
            _ref_revision = self._parsed_refs.get(revision)
            if _ref_revision:  # and _ref_revision[1] in ['H', 'RH', 'T']:
                return _ref_revision[0]

            _tags_shas = self.tags.values()
            # maybe it's a tag ? we don't have them in self.revisions
            if revision in _tags_shas:
                return _tags_shas[_tags_shas.index(revision)]

            elif not SHA_PATTERN.match(revision) or revision not in self.revisions:
                raise ChangesetDoesNotExistError("Revision %s does not exist "
                    "for this repository" % (revision))

        # Ensure we return full id
        if not SHA_PATTERN.match(str(revision)):
            raise ChangesetDoesNotExistError("Given revision %s not recognized"
                % revision)
        return revision

    def _get_archives(self, archive_name='tip'):

        for i in [('zip', '.zip'), ('gz', '.tar.gz'), ('bz2', '.tar.bz2')]:
                yield {"type": i[0], "extension": i[1], "node": archive_name}

    def _get_url(self, url):
        """
        Returns normalized url. If schema is not given, would fall to
        filesystem (``file:///``) schema.
        """
        url = str(url)
        if url != 'default' and not '://' in url:
            url = ':///'.join(('file', url))
        return url

    def get_hook_location(self):
        """
        returns absolute path to location where hooks are stored
        """
        loc = os.path.join(self.path, 'hooks')
        if not self.bare:
            loc = os.path.join(self.path, '.git', 'hooks')
        return loc

    @LazyProperty
    def name(self):
        return os.path.basename(self.path)

    @LazyProperty
    def last_change(self):
        """
        Returns last change made on this repository as datetime object
        """
        return date_fromtimestamp(self._get_mtime(), makedate()[1])

    def _get_mtime(self):
        try:
            return time.mktime(self.get_changeset().date.timetuple())
        except RepositoryError:
            idx_loc = '' if self.bare else '.git'
            # fallback to filesystem
            in_path = os.path.join(self.path, idx_loc, "index")
            he_path = os.path.join(self.path, idx_loc, "HEAD")
            if os.path.exists(in_path):
                return os.stat(in_path).st_mtime
            else:
                return os.stat(he_path).st_mtime

    @LazyProperty
    def description(self):
        idx_loc = '' if self.bare else '.git'
        undefined_description = u'unknown'
        description_path = os.path.join(self.path, idx_loc, 'description')
        if os.path.isfile(description_path):
            return safe_unicode(open(description_path).read())
        else:
            return undefined_description

    @LazyProperty
    def contact(self):
        undefined_contact = u'Unknown'
        return undefined_contact

    @property
    def branches(self):
        if not self.revisions:
            return {}
        sortkey = lambda ctx: ctx[0]
        _branches = [(x[0], x[1][0])
                     for x in self._parsed_refs.iteritems() if x[1][1] == 'H']
        return OrderedDict(sorted(_branches, key=sortkey, reverse=False))

    @LazyProperty
    def tags(self):
        return self._get_tags()

    def _get_tags(self):
        if not self.revisions:
            return {}

        sortkey = lambda ctx: ctx[0]
        _tags = [(x[0], x[1][0])
                 for x in self._parsed_refs.iteritems() if x[1][1] == 'T']
        return OrderedDict(sorted(_tags, key=sortkey, reverse=True))

    def tag(self, name, user, revision=None, message=None, date=None,
            **kwargs):
        """
        Creates and returns a tag for the given ``revision``.

        :param name: name for new tag
        :param user: full username, i.e.: "Joe Doe <joe.doe@example.com>"
        :param revision: changeset id for which new tag would be created
        :param message: message of the tag's commit
        :param date: date of tag's commit

        :raises TagAlreadyExistError: if tag with same name already exists
        """
        if name in self.tags:
            raise TagAlreadyExistError("Tag %s already exists" % name)
        changeset = self.get_changeset(revision)
        message = message or "Added tag %s for commit %s" % (name,
            changeset.raw_id)
        self._repo.refs["refs/tags/%s" % name] = changeset._commit.id

        self._parsed_refs = self._get_parsed_refs()
        self.tags = self._get_tags()
        return changeset

    def remove_tag(self, name, user, message=None, date=None):
        """
        Removes tag with the given ``name``.

        :param name: name of the tag to be removed
        :param user: full username, i.e.: "Joe Doe <joe.doe@example.com>"
        :param message: message of the tag's removal commit
        :param date: date of tag's removal commit

        :raises TagDoesNotExistError: if tag with given name does not exists
        """
        if name not in self.tags:
            raise TagDoesNotExistError("Tag %s does not exist" % name)
        tagpath = posixpath.join(self._repo.refs.path, 'refs', 'tags', name)
        try:
            os.remove(tagpath)
            self._parsed_refs = self._get_parsed_refs()
            self.tags = self._get_tags()
        except OSError, e:
            raise RepositoryError(e.strerror)

    @LazyProperty
    def _parsed_refs(self):
        return self._get_parsed_refs()

    def _get_parsed_refs(self):
        # cache the property
        _repo = self._repo
        refs = _repo.get_refs()
        keys = [('refs/heads/', 'H'),
                ('refs/remotes/origin/', 'RH'),
                ('refs/tags/', 'T')]
        _refs = {}
        for ref, sha in refs.iteritems():
            for k, type_ in keys:
                if ref.startswith(k):
                    _key = ref[len(k):]
                    if type_ == 'T':
                        obj = _repo.get_object(sha)
                        if isinstance(obj, Tag):
                            sha = _repo.get_object(sha).object[1]
                    _refs[_key] = [sha, type_]
                    break
        return _refs

    def _heads(self, reverse=False):
        refs = self._repo.get_refs()
        heads = {}

        for key, val in refs.items():
            for ref_key in ['refs/heads/', 'refs/remotes/origin/']:
                if key.startswith(ref_key):
                    n = key[len(ref_key):]
                    if n not in ['HEAD']:
                        heads[n] = val

        return heads if reverse else dict((y, x) for x, y in heads.iteritems())

    def get_changeset(self, revision=None):
        """
        Returns ``GitChangeset`` object representing commit from git repository
        at the given revision or head (most recent commit) if None given.
        """
        if isinstance(revision, GitChangeset):
            return revision
        revision = self._get_revision(revision)
        changeset = GitChangeset(repository=self, revision=revision)
        return changeset

    def get_changesets(self, start=None, end=None, start_date=None,
           end_date=None, branch_name=None, reverse=False):
        """
        Returns iterator of ``GitChangeset`` objects from start to end (both
        are inclusive), in ascending date order (unless ``reverse`` is set).

        :param start: changeset ID, as str; first returned changeset
        :param end: changeset ID, as str; last returned changeset
        :param start_date: if specified, changesets with commit date less than
          ``start_date`` would be filtered out from returned set
        :param end_date: if specified, changesets with commit date greater than
          ``end_date`` would be filtered out from returned set
        :param branch_name: if specified, changesets not reachable from given
          branch would be filtered out from returned set
        :param reverse: if ``True``, returned generator would be reversed
          (meaning that returned changesets would have descending date order)

        :raise BranchDoesNotExistError: If given ``branch_name`` does not
            exist.
        :raise ChangesetDoesNotExistError: If changeset for given ``start`` or
          ``end`` could not be found.

        """
        if branch_name and branch_name not in self.branches:
            raise BranchDoesNotExistError("Branch '%s' not found" \
                                          % branch_name)
        # %H at format means (full) commit hash, initial hashes are retrieved
        # in ascending date order
        cmd_template = 'log --date-order --reverse --pretty=format:"%H"'
        cmd_params = {}
        if start_date:
            cmd_template += ' --since "$since"'
            cmd_params['since'] = start_date.strftime('%m/%d/%y %H:%M:%S')
        if end_date:
            cmd_template += ' --until "$until"'
            cmd_params['until'] = end_date.strftime('%m/%d/%y %H:%M:%S')
        if branch_name:
            cmd_template += ' $branch_name'
            cmd_params['branch_name'] = branch_name
        else:
            rev_filter = _git_path = settings.GIT_REV_FILTER
            cmd_template += ' %s' % (rev_filter)

        cmd = string.Template(cmd_template).safe_substitute(**cmd_params)
        revs = self.run_git_command(cmd)[0].splitlines()
        start_pos = 0
        end_pos = len(revs)
        if start:
            _start = self._get_revision(start)
            try:
                start_pos = revs.index(_start)
            except ValueError:
                pass

        if end is not None:
            _end = self._get_revision(end)
            try:
                end_pos = revs.index(_end)
            except ValueError:
                pass

        if None not in [start, end] and start_pos > end_pos:
            raise RepositoryError('start cannot be after end')

        if end_pos is not None:
            end_pos += 1

        revs = revs[start_pos:end_pos]
        if reverse:
            revs = reversed(revs)
        return CollectionGenerator(self, revs)

    def get_diff(self, rev1, rev2, path=None, ignore_whitespace=False,
                 context=3):
        """
        Returns (git like) *diff*, as plain text. Shows changes introduced by
        ``rev2`` since ``rev1``.

        :param rev1: Entry point from which diff is shown. Can be
          ``self.EMPTY_CHANGESET`` - in this case, patch showing all
          the changes since empty state of the repository until ``rev2``
        :param rev2: Until which revision changes should be shown.
        :param ignore_whitespace: If set to ``True``, would not show whitespace
          changes. Defaults to ``False``.
        :param context: How many lines before/after changed lines should be
          shown. Defaults to ``3``.
        """
        flags = ['-U%s' % context, '--full-index', '--binary', '-p', '-M', '--abbrev=40']
        if ignore_whitespace:
            flags.append('-w')

        if hasattr(rev1, 'raw_id'):
            rev1 = getattr(rev1, 'raw_id')

        if hasattr(rev2, 'raw_id'):
            rev2 = getattr(rev2, 'raw_id')

        if rev1 == self.EMPTY_CHANGESET:
            rev2 = self.get_changeset(rev2).raw_id
            cmd = ' '.join(['show'] + flags + [rev2])
        else:
            rev1 = self.get_changeset(rev1).raw_id
            rev2 = self.get_changeset(rev2).raw_id
            cmd = ' '.join(['diff'] + flags + [rev1, rev2])

        if path:
            cmd += ' -- "%s"' % path

        stdout, stderr = self.run_git_command(cmd)
        # If we used 'show' command, strip first few lines (until actual diff
        # starts)
        if rev1 == self.EMPTY_CHANGESET:
            lines = stdout.splitlines()
            x = 0
            for line in lines:
                if line.startswith('diff'):
                    break
                x += 1
            # Append new line just like 'diff' command do
            stdout = '\n'.join(lines[x:]) + '\n'
        return stdout

    @LazyProperty
    def in_memory_changeset(self):
        """
        Returns ``GitInMemoryChangeset`` object for this repository.
        """
        return GitInMemoryChangeset(self)

    def clone(self, url, update_after_clone=True, bare=False):
        """
        Tries to clone changes from external location.

        :param update_after_clone: If set to ``False``, git won't checkout
          working directory
        :param bare: If set to ``True``, repository would be cloned into
          *bare* git repository (no working directory at all).
        """
        url = self._get_url(url)
        cmd = ['clone']
        if bare:
            cmd.append('--bare')
        elif not update_after_clone:
            cmd.append('--no-checkout')
        cmd += ['--', '"%s"' % url, '"%s"' % self.path]
        cmd = ' '.join(cmd)
        # If error occurs run_git_command raises RepositoryError already
        self.run_git_command(cmd)

    def pull(self, url):
        """
        Tries to pull changes from external location.
        """
        url = self._get_url(url)
        cmd = ['pull']
        cmd.append("--ff-only")
        cmd.append(url)
        cmd = ' '.join(cmd)
        # If error occurs run_git_command raises RepositoryError already
        self.run_git_command(cmd)

    def fetch(self, url):
        """
        Tries to pull changes from external location.
        """
        url = self._get_url(url)
        so, se = self.run_git_command('ls-remote -h %s' % url)
        refs = []
        for line in (x for x in so.splitlines()):
            sha, ref = line.split('\t')
            refs.append(ref)
        refs = ' '.join(('+%s:%s' % (r, r) for r in refs))
        cmd = '''fetch %s -- %s''' % (url, refs)
        self.run_git_command(cmd)

    @LazyProperty
    def workdir(self):
        """
        Returns ``Workdir`` instance for this repository.
        """
        return GitWorkdir(self)

    def get_config_value(self, section, name, config_file=None):
        """
        Returns configuration value for a given [``section``] and ``name``.

        :param section: Section we want to retrieve value from
        :param name: Name of configuration we want to retrieve
        :param config_file: A path to file which should be used to retrieve
          configuration from (might also be a list of file paths)
        """
        if config_file is None:
            config_file = []
        elif isinstance(config_file, basestring):
            config_file = [config_file]

        def gen_configs():
            for path in config_file + self._config_files:
                try:
                    yield ConfigFile.from_path(path)
                except (IOError, OSError, ValueError):
                    continue

        for config in gen_configs():
            try:
                return config.get(section, name)
            except KeyError:
                continue
        return None

    def get_user_name(self, config_file=None):
        """
        Returns user's name from global configuration file.

        :param config_file: A path to file which should be used to retrieve
          configuration from (might also be a list of file paths)
        """
        return self.get_config_value('user', 'name', config_file)

    def get_user_email(self, config_file=None):
        """
        Returns user's email from global configuration file.

        :param config_file: A path to file which should be used to retrieve
          configuration from (might also be a list of file paths)
        """
        return self.get_config_value('user', 'email', config_file)

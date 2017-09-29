import os
import posixpath

from vcs.conf import settings
from vcs.backends.base import BaseChangeset
from vcs.exceptions import (
    ChangesetDoesNotExistError, ChangesetError, ImproperArchiveTypeError,
    NodeDoesNotExistError, VCSError
)
from vcs.nodes import (
    AddedFileNodesGenerator, ChangedFileNodesGenerator, DirNode, FileNode,
    NodeKind, RemovedFileNodesGenerator, RootNode, SubModuleNode
)
from vcs.utils import safe_str, safe_unicode, date_fromtimestamp
from vcs.utils.lazy import LazyProperty
from vcs.utils.paths import get_dirs_for_path
from vcs.utils.hgcompat import archival, hex


class MercurialChangeset(BaseChangeset):
    """
    Represents state of the repository at the single revision.
    """

    def __init__(self, repository, revision):
        self.repository = repository
        self.raw_id = revision
        self._ctx = repository._repo[revision]
        self.revision = self._ctx._rev
        self.nodes = {}

    @LazyProperty
    def tags(self):
        return map(safe_unicode, self._ctx.tags())

    @LazyProperty
    def branch(self):
        return  safe_unicode(self._ctx.branch())

    @LazyProperty
    def bookmarks(self):
        return map(safe_unicode, self._ctx.bookmarks())

    @LazyProperty
    def message(self):
        return safe_unicode(self._ctx.description())

    @LazyProperty
    def committer(self):
        return safe_unicode(self.author)

    @LazyProperty
    def author(self):
        return safe_unicode(self._ctx.user())

    @LazyProperty
    def date(self):
        return date_fromtimestamp(*self._ctx.date())

    @LazyProperty
    def _timestamp(self):
        return self._ctx.date()[0]

    @LazyProperty
    def status(self):
        """
        Returns modified, added, removed, deleted files for current changeset
        """
        return self.repository._repo.status(self._ctx.p1().node(),
                                            self._ctx.node())

    @LazyProperty
    def _file_paths(self):
        return list(self._ctx)

    @LazyProperty
    def _dir_paths(self):
        p = list(set(get_dirs_for_path(*self._file_paths)))
        p.insert(0, '')
        return p

    @LazyProperty
    def _paths(self):
        return self._dir_paths + self._file_paths

    @LazyProperty
    def id(self):
        if self.last:
            return u'tip'
        return self.short_id

    @LazyProperty
    def short_id(self):
        return self.raw_id[:12]

    @LazyProperty
    def parents(self):
        """
        Returns list of parents changesets.
        """
        return [self.repository.get_changeset(parent.rev())
                for parent in self._ctx.parents() if parent.rev() >= 0]

    @LazyProperty
    def children(self):
        """
        Returns list of children changesets.
        """
        return [self.repository.get_changeset(child.rev())
                for child in self._ctx.children() if child.rev() >= 0]

    def next(self, branch=None):

        if branch and self.branch != branch:
            raise VCSError('Branch option used on changeset not belonging '
                           'to that branch')

        def _next(changeset, branch):
            try:
                next_ = changeset.revision + 1
                next_rev = changeset.repository.revisions[next_]
            except IndexError:
                raise ChangesetDoesNotExistError
            cs = changeset.repository.get_changeset(next_rev)

            if branch and branch != cs.branch:
                return _next(cs, branch)

            return cs

        return _next(self, branch)

    def prev(self, branch=None):
        if branch and self.branch != branch:
            raise VCSError('Branch option used on changeset not belonging '
                           'to that branch')

        def _prev(changeset, branch):
            try:
                prev_ = changeset.revision - 1
                if prev_ < 0:
                    raise IndexError
                prev_rev = changeset.repository.revisions[prev_]
            except IndexError:
                raise ChangesetDoesNotExistError

            cs = changeset.repository.get_changeset(prev_rev)

            if branch and branch != cs.branch:
                return _prev(cs, branch)

            return cs

        return _prev(self, branch)

    def diff(self, ignore_whitespace=True, context=3):
        return ''.join(self._ctx.diff(git=True,
                                      ignore_whitespace=ignore_whitespace,
                                      context=context))

    def _fix_path(self, path):
        """
        Paths are stored without trailing slash so we need to get rid off it if
        needed. Also mercurial keeps filenodes as str so we need to decode
        from unicode to str
        """
        if path.endswith('/'):
            path = path.rstrip('/')

        return safe_str(path)

    def _get_kind(self, path):
        path = self._fix_path(path)
        if path in self._file_paths:
            return NodeKind.FILE
        elif path in self._dir_paths:
            return NodeKind.DIR
        else:
            raise ChangesetError("Node does not exist at the given path '%s'"
                % (path))

    def _get_filectx(self, path):
        path = self._fix_path(path)
        if self._get_kind(path) != NodeKind.FILE:
            raise ChangesetError("File does not exist for revision %s at "
                " '%s'" % (self.raw_id, path))
        return self._ctx.filectx(path)

    def _extract_submodules(self):
        """
        returns a dictionary with submodule information from substate file
        of hg repository
        """
        return self._ctx.substate

    def get_file_mode(self, path):
        """
        Returns stat mode of the file at the given ``path``.
        """
        fctx = self._get_filectx(path)
        if 'x' in fctx.flags():
            return 0100755
        else:
            return 0100644

    def get_file_content(self, path):
        """
        Returns content of the file at given ``path``.
        """
        fctx = self._get_filectx(path)
        return fctx.data()

    def get_file_size(self, path):
        """
        Returns size of the file at given ``path``.
        """
        fctx = self._get_filectx(path)
        return fctx.size()

    def get_file_changeset(self, path):
        """
        Returns last commit of the file at the given ``path``.
        """
        return self.get_file_history(path, limit=1)[0]

    def get_file_history(self, path, limit=None):
        """
        Returns history of file as reversed list of ``Changeset`` objects for
        which file at given ``path`` has been modified.
        """
        fctx = self._get_filectx(path)
        hist = []
        cnt = 0
        for cs in reversed([x for x in fctx.filelog()]):
            cnt += 1
            hist.append(hex(fctx.filectx(cs).node()))
            if limit and cnt == limit:
                break

        return [self.repository.get_changeset(node) for node in hist]

    def get_file_annotate(self, path):
        """
        Returns a generator of four element tuples with
            lineno, sha, changeset lazy loader and line
        """

        fctx = self._get_filectx(path)
        for i, annotate_data in enumerate(fctx.annotate()):
            ln_no = i + 1
            sha = hex(annotate_data[0].node())
            yield (ln_no, sha, lambda: self.repository.get_changeset(sha), annotate_data[1],)

    def fill_archive(self, stream=None, kind='tgz', prefix=None,
                     subrepos=False):
        """
        Fills up given stream.

        :param stream: file like object.
        :param kind: one of following: ``zip``, ``tgz`` or ``tbz2``.
            Default: ``tgz``.
        :param prefix: name of root directory in archive.
            Default is repository name and changeset's raw_id joined with dash
            (``repo-tip.<KIND>``).
        :param subrepos: include subrepos in this archive.

        :raise ImproperArchiveTypeError: If given kind is wrong.
        :raise VcsError: If given stream is None
        """

        allowed_kinds = settings.ARCHIVE_SPECS.keys()
        if kind not in allowed_kinds:
            raise ImproperArchiveTypeError('Archive kind not supported use one'
                'of %s', allowed_kinds)

        if stream is None:
            raise VCSError('You need to pass in a valid stream for filling'
                           ' with archival data')

        if prefix is None:
            prefix = '%s-%s' % (self.repository.name, self.short_id)
        elif prefix.startswith('/'):
            raise VCSError("Prefix cannot start with leading slash")
        elif prefix.strip() == '':
            raise VCSError("Prefix cannot be empty")

        archival.archive(self.repository._repo, stream, self.raw_id,
                         kind, prefix=prefix, subrepos=subrepos)

        if stream.closed and hasattr(stream, 'name'):
            stream = open(stream.name, 'rb')
        elif hasattr(stream, 'mode') and 'r' not in stream.mode:
            stream = open(stream.name, 'rb')
        else:
            stream.seek(0)

    def get_nodes(self, path):
        """
        Returns combined ``DirNode`` and ``FileNode`` objects list representing
        state of changeset at the given ``path``. If node at the given ``path``
        is not instance of ``DirNode``, ChangesetError would be raised.
        """

        if self._get_kind(path) != NodeKind.DIR:
            raise ChangesetError("Directory does not exist for revision %s at "
                " '%s'" % (self.revision, path))
        path = self._fix_path(path)

        filenodes = [FileNode(f, changeset=self) for f in self._file_paths
            if os.path.dirname(f) == path]
        dirs = path == '' and '' or [d for d in self._dir_paths
            if d and posixpath.dirname(d) == path]
        dirnodes = [DirNode(d, changeset=self) for d in dirs
            if os.path.dirname(d) == path]

        als = self.repository.alias
        for k, vals in self._extract_submodules().iteritems():
            #vals = url,rev,type
            loc = vals[0]
            cs = vals[1]
            dirnodes.append(SubModuleNode(k, url=loc, changeset=cs,
                                          alias=als))
        nodes = dirnodes + filenodes
        # cache nodes
        for node in nodes:
            self.nodes[node.path] = node
        nodes.sort()

        return nodes

    def get_node(self, path):
        """
        Returns ``Node`` object from the given ``path``. If there is no node at
        the given ``path``, ``ChangesetError`` would be raised.
        """

        path = self._fix_path(path)

        if not path in self.nodes:
            if path in self._file_paths:
                node = FileNode(path, changeset=self)
            elif path in self._dir_paths or path in self._dir_paths:
                if path == '':
                    node = RootNode(changeset=self)
                else:
                    node = DirNode(path, changeset=self)
            else:
                raise NodeDoesNotExistError("There is no file nor directory "
                    "at the given path: '%s' at revision %s"
                    % (path, self.short_id))
            # cache node
            self.nodes[path] = node
        return self.nodes[path]

    @LazyProperty
    def affected_files(self):
        """
        Get's a fast accessible file changes for given changeset
        """
        return self._ctx.files()

    @property
    def added(self):
        """
        Returns list of added ``FileNode`` objects.
        """
        return AddedFileNodesGenerator([n for n in self.status[1]], self)

    @property
    def changed(self):
        """
        Returns list of modified ``FileNode`` objects.
        """
        return ChangedFileNodesGenerator([n for n in  self.status[0]], self)

    @property
    def removed(self):
        """
        Returns list of removed ``FileNode`` objects.
        """
        return RemovedFileNodesGenerator([n for n in self.status[2]], self)

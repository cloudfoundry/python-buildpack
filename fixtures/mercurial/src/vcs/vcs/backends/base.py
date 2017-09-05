# -*- coding: utf-8 -*-
"""
    vcs.backends.base
    ~~~~~~~~~~~~~~~~~

    Base for all available scm backends

    :created_on: Apr 8, 2010
    :copyright: (c) 2010-2011 by Marcin Kuzminski, Lukasz Balcerzak.
"""

import datetime
import itertools

from vcs.utils import author_name, author_email
from vcs.utils.lazy import LazyProperty
from vcs.utils.helpers import get_dict_for_attrs
from vcs.conf import settings

from vcs.exceptions import (
    ChangesetError, EmptyRepositoryError, NodeAlreadyAddedError,
    NodeAlreadyChangedError, NodeAlreadyExistsError, NodeAlreadyRemovedError,
    NodeDoesNotExistError, NodeNotChangedError, RepositoryError
)


class BaseRepository(object):
    """
    Base Repository for final backends

    **Attributes**

        ``DEFAULT_BRANCH_NAME``
            name of default branch (i.e. "trunk" for svn, "master" for git etc.

        ``scm``
            alias of scm, i.e. *git* or *hg*

        ``repo``
            object from external api

        ``revisions``
            list of all available revisions' ids, in ascending order

        ``changesets``
            storage dict caching returned changesets

        ``path``
            absolute path to the repository

        ``branches``
            branches as list of changesets

        ``tags``
            tags as list of changesets
    """
    scm = None
    DEFAULT_BRANCH_NAME = None
    EMPTY_CHANGESET = '0' * 40

    def __init__(self, repo_path, create=False, **kwargs):
        """
        Initializes repository. Raises RepositoryError if repository could
        not be find at the given ``repo_path`` or directory at ``repo_path``
        exists and ``create`` is set to True.

        :param repo_path: local path of the repository
        :param create=False: if set to True, would try to craete repository.
        :param src_url=None: if set, should be proper url from which repository
          would be cloned; requires ``create`` parameter to be set to True -
          raises RepositoryError if src_url is set and create evaluates to
          False
        """
        raise NotImplementedError

    def __str__(self):
        return '<%s at %s>' % (self.__class__.__name__, self.path)

    def __repr__(self):
        return self.__str__()

    def __len__(self):
        return self.count()

    def __eq__(self, other):
        same_instance = isinstance(other, self.__class__)
        return same_instance and getattr(other, 'path', None) == self.path

    def __ne__(self, other):
        return not self.__eq__(other)

    @LazyProperty
    def alias(self):
        for k, v in settings.BACKENDS.items():
            if v.split('.')[-1] == str(self.__class__.__name__):
                return k

    @LazyProperty
    def name(self):
        raise NotImplementedError

    @LazyProperty
    def owner(self):
        raise NotImplementedError

    @LazyProperty
    def description(self):
        raise NotImplementedError

    @LazyProperty
    def size(self):
        """
        Returns combined size in bytes for all repository files
        """

        size = 0
        try:
            tip = self.get_changeset()
            for topnode, dirs, files in tip.walk('/'):
                for f in files:
                    size += tip.get_file_size(f.path)
                for dir in dirs:
                    for f in files:
                        size += tip.get_file_size(f.path)

        except RepositoryError, e:
            pass
        return size

    def is_valid(self):
        """
        Validates repository.
        """
        raise NotImplementedError

    def get_last_change(self):
        self.get_changesets()

    #==========================================================================
    # CHANGESETS
    #==========================================================================

    def get_changeset(self, revision=None):
        """
        Returns instance of ``Changeset`` class. If ``revision`` is None, most
        recent changeset is returned.

        :raises ``EmptyRepositoryError``: if there are no revisions
        """
        raise NotImplementedError

    def __iter__(self):
        """
        Allows Repository objects to be iterated.

        *Requires* implementation of ``__getitem__`` method.
        """
        for revision in self.revisions:
            yield self.get_changeset(revision)

    def get_changesets(self, start=None, end=None, start_date=None,
                       end_date=None, branch_name=None, reverse=False):
        """
        Returns iterator of ``MercurialChangeset`` objects from start to end
        not inclusive This should behave just like a list, ie. end is not
        inclusive

        :param start: None or str
        :param end: None or str
        :param start_date:
        :param end_date:
        :param branch_name:
        :param reversed:
        """
        raise NotImplementedError

    def __getslice__(self, i, j):
        """
        Returns a iterator of sliced repository
        """
        for rev in self.revisions[i:j]:
            yield self.get_changeset(rev)

    def __getitem__(self, key):
        return self.get_changeset(key)

    def count(self):
        return len(self.revisions)

    def tag(self, name, user, revision=None, message=None, date=None, **opts):
        """
        Creates and returns a tag for the given ``revision``.

        :param name: name for new tag
        :param user: full username, i.e.: "Joe Doe <joe.doe@example.com>"
        :param revision: changeset id for which new tag would be created
        :param message: message of the tag's commit
        :param date: date of tag's commit

        :raises TagAlreadyExistError: if tag with same name already exists
        """
        raise NotImplementedError

    def remove_tag(self, name, user, message=None, date=None):
        """
        Removes tag with the given ``name``.

        :param name: name of the tag to be removed
        :param user: full username, i.e.: "Joe Doe <joe.doe@example.com>"
        :param message: message of the tag's removal commit
        :param date: date of tag's removal commit

        :raises TagDoesNotExistError: if tag with given name does not exists
        """
        raise NotImplementedError

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
        raise NotImplementedError

    # ========== #
    # COMMIT API #
    # ========== #

    @LazyProperty
    def in_memory_changeset(self):
        """
        Returns ``InMemoryChangeset`` object for this repository.
        """
        raise NotImplementedError

    def add(self, filenode, **kwargs):
        """
        Commit api function that will add given ``FileNode`` into this
        repository.

        :raises ``NodeAlreadyExistsError``: if there is a file with same path
          already in repository
        :raises ``NodeAlreadyAddedError``: if given node is already marked as
          *added*
        """
        raise NotImplementedError

    def remove(self, filenode, **kwargs):
        """
        Commit api function that will remove given ``FileNode`` into this
        repository.

        :raises ``EmptyRepositoryError``: if there are no changesets yet
        :raises ``NodeDoesNotExistError``: if there is no file with given path
        """
        raise NotImplementedError

    def commit(self, message, **kwargs):
        """
        Persists current changes made on this repository and returns newly
        created changeset.

        :raises ``NothingChangedError``: if no changes has been made
        """
        raise NotImplementedError

    def get_state(self):
        """
        Returns dictionary with ``added``, ``changed`` and ``removed`` lists
        containing ``FileNode`` objects.
        """
        raise NotImplementedError

    def get_config_value(self, section, name, config_file=None):
        """
        Returns configuration value for a given [``section``] and ``name``.

        :param section: Section we want to retrieve value from
        :param name: Name of configuration we want to retrieve
        :param config_file: A path to file which should be used to retrieve
          configuration from (might also be a list of file paths)
        """
        raise NotImplementedError

    def get_user_name(self, config_file=None):
        """
        Returns user's name from global configuration file.

        :param config_file: A path to file which should be used to retrieve
          configuration from (might also be a list of file paths)
        """
        raise NotImplementedError

    def get_user_email(self, config_file=None):
        """
        Returns user's email from global configuration file.

        :param config_file: A path to file which should be used to retrieve
          configuration from (might also be a list of file paths)
        """
        raise NotImplementedError

    # =========== #
    # WORKDIR API #
    # =========== #

    @LazyProperty
    def workdir(self):
        """
        Returns ``Workdir`` instance for this repository.
        """
        raise NotImplementedError


class BaseChangeset(object):
    """
    Each backend should implement it's changeset representation.

    **Attributes**

        ``repository``
            repository object within which changeset exists

        ``id``
            may be ``raw_id`` or i.e. for mercurial's tip just ``tip``

        ``raw_id``
            raw changeset representation (i.e. full 40 length sha for git
            backend)

        ``short_id``
            shortened (if apply) version of ``raw_id``; it would be simple
            shortcut for ``raw_id[:12]`` for git/mercurial backends or same
            as ``raw_id`` for subversion

        ``revision``
            revision number as integer

        ``files``
            list of ``FileNode`` (``Node`` with NodeKind.FILE) objects

        ``dirs``
            list of ``DirNode`` (``Node`` with NodeKind.DIR) objects

        ``nodes``
            combined list of ``Node`` objects

        ``author``
            author of the changeset, as unicode

        ``message``
            message of the changeset, as unicode

        ``parents``
            list of parent changesets

        ``last``
            ``True`` if this is last changeset in repository, ``False``
            otherwise; trying to access this attribute while there is no
            changesets would raise ``EmptyRepositoryError``
    """
    def __str__(self):
        return '<%s at %s:%s>' % (self.__class__.__name__, self.revision,
            self.short_id)

    def __repr__(self):
        return self.__str__()

    def __unicode__(self):
        return u'%s:%s' % (self.revision, self.short_id)

    def __eq__(self, other):
        return self.raw_id == other.raw_id

    @LazyProperty
    def last(self):
        if self.repository is None:
            raise ChangesetError("Cannot check if it's most recent revision")
        return self.raw_id == self.repository.revisions[-1]

    @LazyProperty
    def parents(self):
        """
        Returns list of parents changesets.
        """
        raise NotImplementedError

    @LazyProperty
    def children(self):
        """
        Returns list of children changesets.
        """
        raise NotImplementedError

    @LazyProperty
    def id(self):
        """
        Returns string identifying this changeset.
        """
        raise NotImplementedError

    @LazyProperty
    def raw_id(self):
        """
        Returns raw string identifying this changeset.
        """
        raise NotImplementedError

    @LazyProperty
    def short_id(self):
        """
        Returns shortened version of ``raw_id`` attribute, as string,
        identifying this changeset, useful for web representation.
        """
        raise NotImplementedError

    @LazyProperty
    def revision(self):
        """
        Returns integer identifying this changeset.

        """
        raise NotImplementedError

    @LazyProperty
    def committer(self):
        """
        Returns Committer for given commit
        """

        raise NotImplementedError

    @LazyProperty
    def committer_name(self):
        """
        Returns Author name for given commit
        """

        return author_name(self.committer)

    @LazyProperty
    def committer_email(self):
        """
        Returns Author email address for given commit
        """

        return author_email(self.committer)

    @LazyProperty
    def author(self):
        """
        Returns Author for given commit
        """

        raise NotImplementedError

    @LazyProperty
    def author_name(self):
        """
        Returns Author name for given commit
        """

        return author_name(self.author)

    @LazyProperty
    def author_email(self):
        """
        Returns Author email address for given commit
        """

        return author_email(self.author)

    def get_file_mode(self, path):
        """
        Returns stat mode of the file at the given ``path``.
        """
        raise NotImplementedError

    def get_file_content(self, path):
        """
        Returns content of the file at the given ``path``.
        """
        raise NotImplementedError

    def get_file_size(self, path):
        """
        Returns size of the file at the given ``path``.
        """
        raise NotImplementedError

    def get_file_changeset(self, path):
        """
        Returns last commit of the file at the given ``path``.
        """
        raise NotImplementedError

    def get_file_history(self, path):
        """
        Returns history of file as reversed list of ``Changeset`` objects for
        which file at given ``path`` has been modified.
        """
        raise NotImplementedError

    def get_nodes(self, path):
        """
        Returns combined ``DirNode`` and ``FileNode`` objects list representing
        state of changeset at the given ``path``.

        :raises ``ChangesetError``: if node at the given ``path`` is not
          instance of ``DirNode``
        """
        raise NotImplementedError

    def get_node(self, path):
        """
        Returns ``Node`` object from the given ``path``.

        :raises ``NodeDoesNotExistError``: if there is no node at the given
          ``path``
        """
        raise NotImplementedError

    def fill_archive(self, stream=None, kind='tgz', prefix=None):
        """
        Fills up given stream.

        :param stream: file like object.
        :param kind: one of following: ``zip``, ``tar``, ``tgz``
            or ``tbz2``. Default: ``tgz``.
        :param prefix: name of root directory in archive.
            Default is repository name and changeset's raw_id joined with dash.

            repo-tip.<kind>
        """

        raise NotImplementedError

    def get_chunked_archive(self, **kwargs):
        """
        Returns iterable archive. Tiny wrapper around ``fill_archive`` method.

        :param chunk_size: extra parameter which controls size of returned
            chunks. Default:8k.
        """

        chunk_size = kwargs.pop('chunk_size', 8192)
        stream = kwargs.get('stream')
        self.fill_archive(**kwargs)
        while True:
            data = stream.read(chunk_size)
            if not data:
                break
            yield data

    @LazyProperty
    def root(self):
        """
        Returns ``RootNode`` object for this changeset.
        """
        return self.get_node('')

    def next(self, branch=None):
        """
        Returns next changeset from current, if branch is gives it will return
        next changeset belonging to this branch

        :param branch: show changesets within the given named branch
        """
        raise NotImplementedError

    def prev(self, branch=None):
        """
        Returns previous changeset from current, if branch is gives it will
        return previous changeset belonging to this branch

        :param branch: show changesets within the given named branch
        """
        raise NotImplementedError

    @LazyProperty
    def added(self):
        """
        Returns list of added ``FileNode`` objects.
        """
        raise NotImplementedError

    @LazyProperty
    def changed(self):
        """
        Returns list of modified ``FileNode`` objects.
        """
        raise NotImplementedError

    @LazyProperty
    def removed(self):
        """
        Returns list of removed ``FileNode`` objects.
        """
        raise NotImplementedError

    @LazyProperty
    def size(self):
        """
        Returns total number of bytes from contents of all filenodes.
        """
        return sum((node.size for node in self.get_filenodes_generator()))

    def walk(self, topurl=''):
        """
        Similar to os.walk method. Insted of filesystem it walks through
        changeset starting at given ``topurl``.  Returns generator of tuples
        (topnode, dirnodes, filenodes).
        """
        topnode = self.get_node(topurl)
        yield (topnode, topnode.dirs, topnode.files)
        for dirnode in topnode.dirs:
            for tup in self.walk(dirnode.path):
                yield tup

    def get_filenodes_generator(self):
        """
        Returns generator that yields *all* file nodes.
        """
        for topnode, dirs, files in self.walk():
            for node in files:
                yield node

    def as_dict(self):
        """
        Returns dictionary with changeset's attributes and their values.
        """
        data = get_dict_for_attrs(self, ['id', 'raw_id', 'short_id',
            'revision', 'date', 'message'])
        data['author'] = {'name': self.author_name, 'email': self.author_email}
        data['added'] = [node.path for node in self.added]
        data['changed'] = [node.path for node in self.changed]
        data['removed'] = [node.path for node in self.removed]
        return data


class BaseWorkdir(object):
    """
    Working directory representation of single repository.

    :attribute: repository: repository object of working directory
    """

    def __init__(self, repository):
        self.repository = repository

    def get_branch(self):
        """
        Returns name of current branch.
        """
        raise NotImplementedError

    def get_changeset(self):
        """
        Returns current changeset.
        """
        raise NotImplementedError

    def get_added(self):
        """
        Returns list of ``FileNode`` objects marked as *new* in working
        directory.
        """
        raise NotImplementedError

    def get_changed(self):
        """
        Returns list of ``FileNode`` objects *changed* in working directory.
        """
        raise NotImplementedError

    def get_removed(self):
        """
        Returns list of ``RemovedFileNode`` objects marked as *removed* in
        working directory.
        """
        raise NotImplementedError

    def get_untracked(self):
        """
        Returns list of ``FileNode`` objects which are present within working
        directory however are not tracked by repository.
        """
        raise NotImplementedError

    def get_status(self):
        """
        Returns dict with ``added``, ``changed``, ``removed`` and ``untracked``
        lists.
        """
        raise NotImplementedError

    def commit(self, message, **kwargs):
        """
        Commits local (from working directory) changes and returns newly
        created
        ``Changeset``. Updates repository's ``revisions`` list.

        :raises ``CommitError``: if any error occurs while committing
        """
        raise NotImplementedError

    def update(self, revision=None):
        """
        Fetches content of the given revision and populates it within working
        directory.
        """
        raise NotImplementedError

    def checkout_branch(self, branch=None):
        """
        Checks out ``branch`` or the backend's default branch.

        Raises ``BranchDoesNotExistError`` if the branch does not exist.
        """
        raise NotImplementedError


class BaseInMemoryChangeset(object):
    """
    Represents differences between repository's state (most recent head) and
    changes made *in place*.

    **Attributes**

        ``repository``
            repository object for this in-memory-changeset

        ``added``
            list of ``FileNode`` objects marked as *added*

        ``changed``
            list of ``FileNode`` objects marked as *changed*

        ``removed``
            list of ``FileNode`` or ``RemovedFileNode`` objects marked to be
            *removed*

        ``parents``
            list of ``Changeset`` representing parents of in-memory changeset.
            Should always be 2-element sequence.

    """

    def __init__(self, repository):
        self.repository = repository
        self.added = []
        self.changed = []
        self.removed = []
        self.parents = []

    def add(self, *filenodes):
        """
        Marks given ``FileNode`` objects as *to be committed*.

        :raises ``NodeAlreadyExistsError``: if node with same path exists at
          latest changeset
        :raises ``NodeAlreadyAddedError``: if node with same path is already
          marked as *added*
        """
        # Check if not already marked as *added* first
        for node in filenodes:
            if node.path in (n.path for n in self.added):
                raise NodeAlreadyAddedError("Such FileNode %s is already "
                    "marked for addition" % node.path)
        for node in filenodes:
            self.added.append(node)

    def change(self, *filenodes):
        """
        Marks given ``FileNode`` objects to be *changed* in next commit.

        :raises ``EmptyRepositoryError``: if there are no changesets yet
        :raises ``NodeAlreadyExistsError``: if node with same path is already
          marked to be *changed*
        :raises ``NodeAlreadyRemovedError``: if node with same path is already
          marked to be *removed*
        :raises ``NodeDoesNotExistError``: if node doesn't exist in latest
          changeset
        :raises ``NodeNotChangedError``: if node hasn't really be changed
        """
        for node in filenodes:
            if node.path in (n.path for n in self.removed):
                raise NodeAlreadyRemovedError("Node at %s is already marked "
                    "as removed" % node.path)
        try:
            self.repository.get_changeset()
        except EmptyRepositoryError:
            raise EmptyRepositoryError("Nothing to change - try to *add* new "
                "nodes rather than changing them")
        for node in filenodes:
            if node.path in (n.path for n in self.changed):
                raise NodeAlreadyChangedError("Node at '%s' is already "
                    "marked as changed" % node.path)
            self.changed.append(node)

    def remove(self, *filenodes):
        """
        Marks given ``FileNode`` (or ``RemovedFileNode``) objects to be
        *removed* in next commit.

        :raises ``NodeAlreadyRemovedError``: if node has been already marked to
          be *removed*
        :raises ``NodeAlreadyChangedError``: if node has been already marked to
          be *changed*
        """
        for node in filenodes:
            if node.path in (n.path for n in self.removed):
                raise NodeAlreadyRemovedError("Node is already marked to "
                    "for removal at %s" % node.path)
            if node.path in (n.path for n in self.changed):
                raise NodeAlreadyChangedError("Node is already marked to "
                    "be changed at %s" % node.path)
            # We only mark node as *removed* - real removal is done by
            # commit method
            self.removed.append(node)

    def reset(self):
        """
        Resets this instance to initial state (cleans ``added``, ``changed``
        and ``removed`` lists).
        """
        self.added = []
        self.changed = []
        self.removed = []
        self.parents = []

    def get_ipaths(self):
        """
        Returns generator of paths from nodes marked as added, changed or
        removed.
        """
        for node in itertools.chain(self.added, self.changed, self.removed):
            yield node.path

    def get_paths(self):
        """
        Returns list of paths from nodes marked as added, changed or removed.
        """
        return list(self.get_ipaths())

    def check_integrity(self, parents=None):
        """
        Checks in-memory changeset's integrity. Also, sets parents if not
        already set.

        :raises CommitError: if any error occurs (i.e.
          ``NodeDoesNotExistError``).
        """
        if not self.parents:
            parents = parents or []
            if len(parents) == 0:
                try:
                    parents = [self.repository.get_changeset(), None]
                except EmptyRepositoryError:
                    parents = [None, None]
            elif len(parents) == 1:
                parents += [None]
            self.parents = parents

        # Local parents, only if not None
        parents = [p for p in self.parents if p]

        # Check nodes marked as added
        for p in parents:
            for node in self.added:
                try:
                    p.get_node(node.path)
                except NodeDoesNotExistError:
                    pass
                else:
                    raise NodeAlreadyExistsError("Node at %s already exists "
                        "at %s" % (node.path, p))

        # Check nodes marked as changed
        missing = set(self.changed)
        not_changed = set(self.changed)
        if self.changed and not parents:
            raise NodeDoesNotExistError(str(self.changed[0].path))
        for p in parents:
            for node in self.changed:
                try:
                    old = p.get_node(node.path)
                    missing.remove(node)
                    if old.content != node.content:
                        not_changed.remove(node)
                except NodeDoesNotExistError:
                    pass
        if self.changed and missing:
            raise NodeDoesNotExistError("Node at %s is missing "
                "(parents: %s)" % (node.path, parents))

        if self.changed and not_changed:
            raise NodeNotChangedError("Node at %s wasn't actually changed "
                "since parents' changesets: %s" % (not_changed.pop().path,
                    parents)
            )

        # Check nodes marked as removed
        if self.removed and not parents:
            raise NodeDoesNotExistError("Cannot remove node at %s as there "
                "were no parents specified" % self.removed[0].path)
        really_removed = set()
        for p in parents:
            for node in self.removed:
                try:
                    p.get_node(node.path)
                    really_removed.add(node)
                except ChangesetError:
                    pass
        not_removed = set(self.removed) - really_removed
        if not_removed:
            raise NodeDoesNotExistError("Cannot remove node at %s from "
                "following parents: %s" % (not_removed[0], parents))

    def commit(self, message, author, parents=None, branch=None, date=None,
            **kwargs):
        """
        Performs in-memory commit (doesn't check workdir in any way) and
        returns newly created ``Changeset``. Updates repository's
        ``revisions``.

        .. note::
            While overriding this method each backend's should call
            ``self.check_integrity(parents)`` in the first place.

        :param message: message of the commit
        :param author: full username, i.e. "Joe Doe <joe.doe@example.com>"
        :param parents: single parent or sequence of parents from which commit
          would be derieved
        :param date: ``datetime.datetime`` instance. Defaults to
          ``datetime.datetime.now()``.
        :param branch: branch name, as string. If none given, default backend's
          branch would be used.

        :raises ``CommitError``: if any error occurs while committing
        """
        raise NotImplementedError


class EmptyChangeset(BaseChangeset):
    """
    An dummy empty changeset. It's possible to pass hash when creating
    an EmptyChangeset
    """

    def __init__(self, cs='0' * 40, repo=None, requested_revision=None,
                 alias=None, revision=-1, message='', author='', date=None):
        self._empty_cs = cs
        self.revision = revision
        self.message = message
        self.author = author
        self.date = date or datetime.datetime.fromtimestamp(0)
        self.repository = repo
        self.requested_revision = requested_revision
        self.alias = alias

    @LazyProperty
    def raw_id(self):
        """
        Returns raw string identifying this changeset, useful for web
        representation.
        """

        return self._empty_cs

    @LazyProperty
    def branch(self):
        from vcs.backends import get_backend
        return get_backend(self.alias).DEFAULT_BRANCH_NAME

    @LazyProperty
    def short_id(self):
        return self.raw_id[:12]

    def get_file_changeset(self, path):
        return self

    def get_file_content(self, path):
        return u''

    def get_file_size(self, path):
        return 0


class CollectionGenerator(object):

    def __init__(self, repo, revs):
        self.repo = repo
        self.revs = revs

    def __len__(self):
        return len(self.revs)

    def __iter__(self):
        for rev in self.revs:
            yield self.repo.get_changeset(rev)

    def __getslice__(self, i, j):
        """
        Returns a iterator of sliced repository
        """
        sliced_revs = self.revs[i:j]
        return CollectionGenerator(self.repo, sliced_revs)

    def __repr__(self):
        return 'CollectionGenerator<%s>' % (len(self))

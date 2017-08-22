# -*- coding: utf-8 -*-
"""
    vcs.nodes
    ~~~~~~~~~

    Module holding everything related to vcs nodes.

    :created_on: Apr 8, 2010
    :copyright: (c) 2010-2011 by Marcin Kuzminski, Lukasz Balcerzak.
"""
import os
import stat
import posixpath
import mimetypes

from pygments import lexers

from vcs.backends.base import EmptyChangeset
from vcs.exceptions import NodeError, RemovedFileNodeError
from vcs.utils.lazy import LazyProperty
from vcs.utils import safe_unicode


class NodeKind:
    SUBMODULE = -1
    DIR = 1
    FILE = 2


class NodeState:
    ADDED = u'added'
    CHANGED = u'changed'
    NOT_CHANGED = u'not changed'
    REMOVED = u'removed'


class NodeGeneratorBase(object):
    """
    Base class for removed added and changed filenodes, it's a lazy generator
    class that will create filenodes only on iteration or call

    The len method doesn't need to create filenodes at all
    """

    def __init__(self, current_paths, cs):
        self.cs = cs
        self.current_paths = current_paths

    def __call__(self):
        return [n for n in self]

    def __getslice__(self, i, j):
        for p in self.current_paths[i:j]:
            yield self.cs.get_node(p)

    def __len__(self):
        return len(self.current_paths)

    def __iter__(self):
        for p in self.current_paths:
            yield self.cs.get_node(p)


class AddedFileNodesGenerator(NodeGeneratorBase):
    """
    Class holding Added files for current changeset
    """
    pass


class ChangedFileNodesGenerator(NodeGeneratorBase):
    """
    Class holding Changed files for current changeset
    """
    pass


class RemovedFileNodesGenerator(NodeGeneratorBase):
    """
    Class holding removed files for current changeset
    """
    def __iter__(self):
        for p in self.current_paths:
            yield RemovedFileNode(path=p)

    def __getslice__(self, i, j):
        for p in self.current_paths[i:j]:
            yield RemovedFileNode(path=p)


class Node(object):
    """
    Simplest class representing file or directory on repository.  SCM backends
    should use ``FileNode`` and ``DirNode`` subclasses rather than ``Node``
    directly.

    Node's ``path`` cannot start with slash as we operate on *relative* paths
    only. Moreover, every single node is identified by the ``path`` attribute,
    so it cannot end with slash, too. Otherwise, path could lead to mistakes.
    """

    def __init__(self, path, kind):
        if path.startswith('/'):
            raise NodeError("Cannot initialize Node objects with slash at "
                "the beginning as only relative paths are supported")
        self.path = path.rstrip('/')
        if path == '' and kind != NodeKind.DIR:
            raise NodeError("Only DirNode and its subclasses may be "
                            "initialized with empty path")
        self.kind = kind
        #self.dirs, self.files = [], []
        if self.is_root() and not self.is_dir():
            raise NodeError("Root node cannot be FILE kind")

    @LazyProperty
    def parent(self):
        parent_path = self.get_parent_path()
        if parent_path:
            if self.changeset:
                return self.changeset.get_node(parent_path)
            return DirNode(parent_path)
        return None

    @LazyProperty
    def unicode_path(self):
        return safe_unicode(self.path)

    @LazyProperty
    def name(self):
        """
        Returns name of the node so if its path
        then only last part is returned.
        """
        return safe_unicode(self.path.rstrip('/').split('/')[-1])

    def _get_kind(self):
        return self._kind

    def _set_kind(self, kind):
        if hasattr(self, '_kind'):
            raise NodeError("Cannot change node's kind")
        else:
            self._kind = kind
            # Post setter check (path's trailing slash)
            if self.path.endswith('/'):
                raise NodeError("Node's path cannot end with slash")

    kind = property(_get_kind, _set_kind)

    def __cmp__(self, other):
        """
        Comparator using name of the node, needed for quick list sorting.
        """
        kind_cmp = cmp(self.kind, other.kind)
        if kind_cmp:
            return kind_cmp
        return cmp(self.name, other.name)

    def __eq__(self, other):
        for attr in ['name', 'path', 'kind']:
            if getattr(self, attr) != getattr(other, attr):
                return False
        if self.is_file():
            if self.content != other.content:
                return False
        else:
            # For DirNode's check without entering each dir
            self_nodes_paths = list(sorted(n.path for n in self.nodes))
            other_nodes_paths = list(sorted(n.path for n in self.nodes))
            if self_nodes_paths != other_nodes_paths:
                return False
        return True

    def __nq__(self, other):
        return not self.__eq__(other)

    def __repr__(self):
        return '<%s %r>' % (self.__class__.__name__, self.path)

    def __str__(self):
        return self.__repr__()

    def __unicode__(self):
        return self.name

    def get_parent_path(self):
        """
        Returns node's parent path or empty string if node is root.
        """
        if self.is_root():
            return ''
        return posixpath.dirname(self.path.rstrip('/')) + '/'

    def is_file(self):
        """
        Returns ``True`` if node's kind is ``NodeKind.FILE``, ``False``
        otherwise.
        """
        return self.kind == NodeKind.FILE

    def is_dir(self):
        """
        Returns ``True`` if node's kind is ``NodeKind.DIR``, ``False``
        otherwise.
        """
        return self.kind == NodeKind.DIR

    def is_root(self):
        """
        Returns ``True`` if node is a root node and ``False`` otherwise.
        """
        return self.kind == NodeKind.DIR and self.path == ''

    def is_submodule(self):
        """
        Returns ``True`` if node's kind is ``NodeKind.SUBMODULE``, ``False``
        otherwise.
        """
        return self.kind == NodeKind.SUBMODULE

    @LazyProperty
    def added(self):
        return self.state is NodeState.ADDED

    @LazyProperty
    def changed(self):
        return self.state is NodeState.CHANGED

    @LazyProperty
    def not_changed(self):
        return self.state is NodeState.NOT_CHANGED

    @LazyProperty
    def removed(self):
        return self.state is NodeState.REMOVED


class FileNode(Node):
    """
    Class representing file nodes.

    :attribute: path: path to the node, relative to repostiory's root
    :attribute: content: if given arbitrary sets content of the file
    :attribute: changeset: if given, first time content is accessed, callback
    :attribute: mode: octal stat mode for a node. Default is 0100644.
    """

    def __init__(self, path, content=None, changeset=None, mode=None):
        """
        Only one of ``content`` and ``changeset`` may be given. Passing both
        would raise ``NodeError`` exception.

        :param path: relative path to the node
        :param content: content may be passed to constructor
        :param changeset: if given, will use it to lazily fetch content
        :param mode: octal representation of ST_MODE (i.e. 0100644)
        """

        if content and changeset:
            raise NodeError("Cannot use both content and changeset")
        super(FileNode, self).__init__(path, kind=NodeKind.FILE)
        self.changeset = changeset
        self._content = content
        self._mode = mode or 0100644

    @LazyProperty
    def mode(self):
        """
        Returns lazily mode of the FileNode. If ``changeset`` is not set, would
        use value given at initialization or 0100644 (default).
        """
        if self.changeset:
            mode = self.changeset.get_file_mode(self.path)
        else:
            mode = self._mode
        return mode

    def _get_content(self):
        if self.changeset:
            content = self.changeset.get_file_content(self.path)
        else:
            content = self._content
        return content

    @property
    def content(self):
        """
        Returns lazily content of the FileNode. If possible, would try to
        decode content from UTF-8.
        """
        content = self._get_content()

        if bool(content and '\0' in content):
            return content
        return safe_unicode(content)

    @LazyProperty
    def size(self):
        if self.changeset:
            return self.changeset.get_file_size(self.path)
        raise NodeError("Cannot retrieve size of the file without related "
            "changeset attribute")

    @LazyProperty
    def message(self):
        if self.changeset:
            return self.last_changeset.message
        raise NodeError("Cannot retrieve message of the file without related "
            "changeset attribute")

    @LazyProperty
    def last_changeset(self):
        if self.changeset:
            return self.changeset.get_file_changeset(self.path)
        raise NodeError("Cannot retrieve last changeset of the file without "
            "related changeset attribute")

    def get_mimetype(self):
        """
        Mimetype is calculated based on the file's content. If ``_mimetype``
        attribute is available, it will be returned (backends which store
        mimetypes or can easily recognize them, should set this private
        attribute to indicate that type should *NOT* be calculated).
        """
        if hasattr(self, '_mimetype'):
            if (isinstance(self._mimetype, (tuple, list,)) and
                len(self._mimetype) == 2):
                return self._mimetype
            else:
                raise NodeError('given _mimetype attribute must be an 2 '
                               'element list or tuple')

        mtype, encoding = mimetypes.guess_type(self.name)

        if mtype is None:
            if self.is_binary:
                mtype = 'application/octet-stream'
                encoding = None
            else:
                mtype = 'text/plain'
                encoding = None
        return mtype, encoding

    @LazyProperty
    def mimetype(self):
        """
        Wrapper around full mimetype info. It returns only type of fetched
        mimetype without the encoding part. use get_mimetype function to fetch
        full set of (type,encoding)
        """
        return self.get_mimetype()[0]

    @LazyProperty
    def mimetype_main(self):
        return self.mimetype.split('/')[0]

    @LazyProperty
    def lexer(self):
        """
        Returns pygment's lexer class. Would try to guess lexer taking file's
        content, name and mimetype.
        """

        try:
            lexer = lexers.guess_lexer_for_filename(self.name, self.content, stripnl=False)
        except lexers.ClassNotFound:
            lexer = lexers.TextLexer(stripnl=False)
        # returns first alias
        return lexer

    @LazyProperty
    def lexer_alias(self):
        """
        Returns first alias of the lexer guessed for this file.
        """
        return self.lexer.aliases[0]

    @LazyProperty
    def history(self):
        """
        Returns a list of changeset for this file in which the file was changed
        """
        if self.changeset is None:
            raise NodeError('Unable to get changeset for this FileNode')
        return self.changeset.get_file_history(self.path)

    @LazyProperty
    def annotate(self):
        """
        Returns a list of three element tuples with lineno,changeset and line
        """
        if self.changeset is None:
            raise NodeError('Unable to get changeset for this FileNode')
        return self.changeset.get_file_annotate(self.path)

    @LazyProperty
    def state(self):
        if not self.changeset:
            raise NodeError("Cannot check state of the node if it's not "
                "linked with changeset")
        elif self.path in (node.path for node in self.changeset.added):
            return NodeState.ADDED
        elif self.path in (node.path for node in self.changeset.changed):
            return NodeState.CHANGED
        else:
            return NodeState.NOT_CHANGED

    @property
    def is_binary(self):
        """
        Returns True if file has binary content.
        """
        _bin = '\0' in self._get_content()
        return _bin

    @LazyProperty
    def extension(self):
        """Returns filenode extension"""
        return self.name.split('.')[-1]

    def is_executable(self):
        """
        Returns ``True`` if file has executable flag turned on.
        """
        return bool(self.mode & stat.S_IXUSR)

    def __repr__(self):
        return '<%s %r @ %s>' % (self.__class__.__name__, self.path,
                                 getattr(self.changeset, 'short_id', ''))


class RemovedFileNode(FileNode):
    """
    Dummy FileNode class - trying to access any public attribute except path,
    name, kind or state (or methods/attributes checking those two) would raise
    RemovedFileNodeError.
    """
    ALLOWED_ATTRIBUTES = [
        'name', 'path', 'state', 'is_root', 'is_file', 'is_dir', 'kind',
        'added', 'changed', 'not_changed', 'removed'
    ]

    def __init__(self, path):
        """
        :param path: relative path to the node
        """
        super(RemovedFileNode, self).__init__(path=path)

    def __getattribute__(self, attr):
        if attr.startswith('_') or attr in RemovedFileNode.ALLOWED_ATTRIBUTES:
            return super(RemovedFileNode, self).__getattribute__(attr)
        raise RemovedFileNodeError("Cannot access attribute %s on "
            "RemovedFileNode" % attr)

    @LazyProperty
    def state(self):
        return NodeState.REMOVED


class DirNode(Node):
    """
    DirNode stores list of files and directories within this node.
    Nodes may be used standalone but within repository context they
    lazily fetch data within same repositorty's changeset.
    """

    def __init__(self, path, nodes=(), changeset=None):
        """
        Only one of ``nodes`` and ``changeset`` may be given. Passing both
        would raise ``NodeError`` exception.

        :param path: relative path to the node
        :param nodes: content may be passed to constructor
        :param changeset: if given, will use it to lazily fetch content
        :param size: always 0 for ``DirNode``
        """
        if nodes and changeset:
            raise NodeError("Cannot use both nodes and changeset")
        super(DirNode, self).__init__(path, NodeKind.DIR)
        self.changeset = changeset
        self._nodes = nodes

    @LazyProperty
    def content(self):
        raise NodeError("%s represents a dir and has no ``content`` attribute"
            % self)

    @LazyProperty
    def nodes(self):
        if self.changeset:
            nodes = self.changeset.get_nodes(self.path)
        else:
            nodes = self._nodes
        self._nodes_dict = dict((node.path, node) for node in nodes)
        return sorted(nodes)

    @LazyProperty
    def files(self):
        return sorted((node for node in self.nodes if node.is_file()))

    @LazyProperty
    def dirs(self):
        return sorted((node for node in self.nodes if node.is_dir()))

    def __iter__(self):
        for node in self.nodes:
            yield node

    def get_node(self, path):
        """
        Returns node from within this particular ``DirNode``, so it is now
        allowed to fetch, i.e. node located at 'docs/api/index.rst' from node
        'docs'. In order to access deeper nodes one must fetch nodes between
        them first - this would work::

           docs = root.get_node('docs')
           docs.get_node('api').get_node('index.rst')

        :param: path - relative to the current node

        .. note::
           To access lazily (as in example above) node have to be initialized
           with related changeset object - without it node is out of
           context and may know nothing about anything else than nearest
           (located at same level) nodes.
        """
        try:
            path = path.rstrip('/')
            if path == '':
                raise NodeError("Cannot retrieve node without path")
            self.nodes  # access nodes first in order to set _nodes_dict
            paths = path.split('/')
            if len(paths) == 1:
                if not self.is_root():
                    path = '/'.join((self.path, paths[0]))
                else:
                    path = paths[0]
                return self._nodes_dict[path]
            elif len(paths) > 1:
                if self.changeset is None:
                    raise NodeError("Cannot access deeper "
                                    "nodes without changeset")
                else:
                    path1, path2 = paths[0], '/'.join(paths[1:])
                    return self.get_node(path1).get_node(path2)
            else:
                raise KeyError
        except KeyError:
            raise NodeError("Node does not exist at %s" % path)

    @LazyProperty
    def state(self):
        raise NodeError("Cannot access state of DirNode")

    @LazyProperty
    def size(self):
        size = 0
        for root, dirs, files in self.changeset.walk(self.path):
            for f in files:
                size += f.size

        return size

    def __repr__(self):
        return '<%s %r @ %s>' % (self.__class__.__name__, self.path,
                                 getattr(self.changeset, 'short_id', ''))


class RootNode(DirNode):
    """
    DirNode being the root node of the repository.
    """

    def __init__(self, nodes=(), changeset=None):
        super(RootNode, self).__init__(path='', nodes=nodes,
            changeset=changeset)

    def __repr__(self):
        return '<%s>' % self.__class__.__name__


class SubModuleNode(Node):
    """
    represents a SubModule of Git or SubRepo of Mercurial
    """
    is_binary = False
    size = 0

    def __init__(self, name, url=None, changeset=None, alias=None):
        self.path = name
        self.kind = NodeKind.SUBMODULE
        self.alias = alias
        # we have to use emptyChangeset here since this can point to svn/git/hg
        # submodules we cannot get from repository
        self.changeset = EmptyChangeset(str(changeset), alias=alias)
        self.url = url or self._extract_submodule_url()

    def __repr__(self):
        return '<%s %r @ %s>' % (self.__class__.__name__, self.path,
                                 getattr(self.changeset, 'short_id', ''))

    def _extract_submodule_url(self):
        if self.alias == 'git':
            #TODO: find a way to parse gits submodule file and extract the
            # linking URL
            return self.path
        if self.alias == 'hg':
            return self.path

    @LazyProperty
    def name(self):
        """
        Returns name of the node so if its path
        then only last part is returned.
        """
        org = safe_unicode(self.path.rstrip('/').split('/')[-1])
        return u'%s @ %s' % (org, self.changeset.short_id)

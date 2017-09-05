.. _quickstart:

Quickstart
==========

Say you don't want to install ``vcs`` or just want to begin with really fast
tutorial?  Not a problem, just follow sections below.

Prepare
-------

We will try to show you how you can use ``vcs`` directly on repository.  But
hey, ``vcs`` is maintained within git `repository
<https://github.com/codeinn/vcs>`_ already, so why not use it? Simply run
following commands in your shell

.. code-block:: bash

   cd /tmp
   git clone git://github.com/codeinn/vcs.git
   cd vcs

Now run your python interpreter of choice::

   $ python
   >>>

.. note::
   You may of course put your clone of ``vcs`` wherever you like but running
   python shell *inside* of it would allow you to use just cloned version of
   ``vcs``.

Take the shortcut
-----------------

There is no need to import everything from ``vcs`` - in fact, all you'd need is
to import ``get_repo``, at least for now. Then, simply initialize repository
object by providing it's type and path.

.. code-block:: python

   >>> import vcs
   >>> # create repository representation at current dir
   >>> repo = vcs.get_repo(path='')

.. note::
   In above example we didn't specify scm. We can provide as second argument to
   the ``get_repo`` function, i.e. ``get_repo('', 'hg')``.

Basics
------

Let's ask repo about the content...

.. code-block:: python

   >>> root = repo.get_changeset().get_node('')
   >>> print root.nodes # prints nodes of the RootNode
   [<DirNode ''>, <DirNode 'docs'>, <DirNode 'tests'>, # ... (chopped)
   >>>
   >>> # get 10th changeset
   >>> chset = repo.get_changeset(10)
   >>> print chset
   <GitChangeset at 10:d955cd312c17>
   >>>
   >>> # any backend would return latest changeset if revision is not given
   >>> tip = repo.get_changeset()
   >>> tip == repo.get_changeset('tip') # for git/mercurial backend 'tip' is allowed
   True
   >>> tip == repo.get_changeset(None) # any backend allow revision to be None (default)
   True
   >>> tip.raw_id == repo.revisions[-1]
   True
   >>>
   >>> # Iterate repository
   >>> for cs in repo:
   ...     print cs
   ...
   ...
   >>> <GitChangeset at 0:c1214f7e79e0>
   >>> <GitChangeset at 1:38b5fe81f109>
   >>> ...

Walking
-------

Now let's ask for nodes at revision faebbb751cc36c137127c50f57bcdb5f1c540013
(https://github.com/codeinn/vcs/commit/faebbb751cc36c137127c50f57bcdb5f1c540013)

.. code-block:: python

   >>> chset = repo.get_changeset('faebbb751cc36c137127c50f57bcdb5f1c540013')
   >>> root = chset.root
   >>> print root.dirs
   [<DirNode 'docs'>, <DirNode 'tests'>, <DirNode 'vcs'>]

.. note::

   :ref:`api-nodes` are objects representing files and directories within the
   repository revision.

.. code-block:: python

   >>> # Fetch vcs directory
   >>> vcs = repo.get_changeset('faebbb751cc36c137127c50f57bcdb5f1c540013').get_node('vcs')
   >>> print vcs.dirs
   [<DirNode 'vcs/backends'>,
    <DirNode 'vcs/utils'>,
    <DirNode 'vcs/web'>]

   >>> backends_node = vcs.dirs[0]
   >>> print backends_node.nodes
   [<FileNode 'vcs/backends/__init__.py'>,
    <FileNode 'vcs/backends/base.py'>,
    <FileNode 'vcs/backends/git.py'>,
    <FileNode 'vcs/backends/hg.py'>]

   >>> print '\n'.join(backends_node.files[0].content.splitlines()[:4])
   # -*- coding: utf-8 -*-
   """
       vcs.backends
       ~~~~~~~~~~~~


Getting meta data
-----------------

Make ``vcs`` show us some meta information

Tags and branches
~~~~~~~~~~~~~~~~~

.. code-block:: python

   >>> print repo.branches
   OrderedDict([('master', 'fe568b4081755c12abf6ba673ba777fc02a415f3')])
   >>> for tag, raw_id in repo.tags.items():
   ...     print tag.rjust(10), '|', raw_id
   ...
    v0.1.9 | 341d28f0eec5ddf0b6b77871e13c2bbd6bec685c
    v0.1.8 | 74ebce002c088b8a5ecf40073db09375515ecd68
    v0.1.7 | 4d78bf73b5c22c82b68f902f138f7881b4fffa2c
    v0.1.6 | 0205cb3f44223fb3099d12a77a69c81b798772d9
    v0.1.5 | 6c0ce52b229aa978889e91b38777f800e85f330b
    v0.1.4 | 7d735150934cd7645ac3051903add952390324a5
    v0.1.3 | 5a3a8fb005554692b16e21dee62bf02667d8dc3e
    v0.1.2 | 0ba5f8a4660034ff25c0cac2a5baabf5d2791d63
   v0.1.11 | c60f01b77c42dce653d6b1d3b04689862c261929
   v0.1.10 | 10cddef6b794696066fb346434014f0a56810218
    v0.1.1 | e6ea6d16e2f26250124a1f4b4fe37a912f9d86a0

Give me a file, finally!
~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: python

   >>> import vcs
   >>> repo = vcs.get_repo('')
   >>> chset = repo.get_changeset('faebbb751cc36c137127c50f57bcdb5f1c540013')
   >>> root = chset.get_node('')
   >>> backends = root.get_node('vcs/backends')
   >>> backends.files
   [<FileNode 'vcs/backends/__init__.py'>,
    <FileNode 'vcs/backends/base.py'>,
    <FileNode 'vcs/backends/git.py'>,
    <FileNode 'vcs/backends/hg.py'>]
   >>> f = backends.get_node('hg.py')
   >>> f.name
   'hg.py'
   >>> f.path
   'vcs/backends/hg.py'
   >>> f.size
   28549
   >>> f.last_changeset
   <GitChangeset at 412:faebbb751cc3>
   >>> f.last_changeset.date
   datetime.datetime(2011, 2, 28, 23, 23, 5)
   >>> f.last_changeset.message
   u'fixed bug in get_changeset when 0 or None was passed'
   >>> f.last_changeset.author
   u'marcinkuzminski <none@none>'
   >>> f.mimetype
   'text/x-python'
   >>> # Following would raise exception unless you have pygments installed
   >>> f.lexer
   <pygments.lexers.PythonLexer>
   >>> f.lexer_alias # shortcut to get first of lexers' available aliases
   'python'
   >>> # wanna go back? why? oh, whatever...
   >>> f.parent
   <DirNode 'vcs/backends'>
   >>>
   >>> # is it cached? Hell yeah...
   >>> f is f.parent.get_node('hg.py') is chset.get_node('vcs/backends/hg.py')
   True

How about history?
~~~~~~~~~~~~~~~~~~

It is possible to retrieve changesets for which file node has been changed and
this is pretty damn simple. Let's say we want to see history of the file located
at ``vcs/nodes.py``.

.. code-block:: python

   >>> f = repo.get_changeset().get_node('vcs/nodes.py')
   >>> for cs in f.history:
   ...      print cs
   ...
   <GitChangeset at 440:40a2d5d71b75>
   <GitChangeset at 438:d1f898326327>
   <GitChangeset at 420:162a36830c23>
   <GitChangeset at 345:c994f0de03b2>
   <GitChangeset at 340:5d3d4d2c262e>
   <GitChangeset at 334:4d4278a6390e>
   <GitChangeset at 298:00dffb625166>
   <GitChangeset at 297:47b6be9a812e>
   <GitChangeset at 289:1589fed841cd>
   <GitChangeset at 285:afafd0ee2821>
   <GitChangeset at 284:639b115ed2b0>
   <GitChangeset at 283:fcf7562d7305>
   <GitChangeset at 256:ec8cbdb5f364>
   <GitChangeset at 255:0d74d2e2bdf3>
   <GitChangeset at 243:6894ad7d8223>
   <GitChangeset at 231:31b3f4b599fa>
   <GitChangeset at 220:3d2515dd21fb>
   <GitChangeset at 186:f804e27aa496>
   <GitChangeset at 182:7f00513785a1>
   <GitChangeset at 181:6efcdc61028c>
   <GitChangeset at 175:6c0ce52b229a>
   <GitChangeset at 165:09788a0b8a54>
   <GitChangeset at 163:0164ee729def>
   <GitChangeset at 140:33fa32233551>
   <GitChangeset at 126:fa014c12c26d>
   <GitChangeset at 111:e686b958768e>
   <GitChangeset at 109:ab5721ca0a08>
   <GitChangeset at 108:c877b68d18e7>
   <GitChangeset at 107:4313566d2e41>
   <GitChangeset at 104:6c2303a79367>
   <GitChangeset at 102:54386793436c>
   <GitChangeset at 101:54000345d2e7>
   <GitChangeset at 99:1c6b3677b37e>
   <GitChangeset at 93:2d03ca750a44>
   <GitChangeset at 92:2a08b128c206>
   <GitChangeset at 91:30c26513ff1e>
   <GitChangeset at 82:ac71e9503c2c>
   <GitChangeset at 81:12669288fd13>
   <GitChangeset at 76:5a0c84f3e6fe>
   <GitChangeset at 73:12f2f5e2b38e>
   <GitChangeset at 61:5eab1222a7cd>
   <GitChangeset at 60:f50f42baeed5>
   <GitChangeset at 59:d7e390a45f6a>
   <GitChangeset at 58:f15c21f97864>
   <GitChangeset at 57:e906ef056cf5>
   <GitChangeset at 56:ea2b108b48aa>
   <GitChangeset at 50:84dec09632a4>
   <GitChangeset at 48:0115510b70c7>
   <GitChangeset at 46:2a13f185e452>
   <GitChangeset at 30:3bf1c5868e57>
   <GitChangeset at 26:b8d040125747>
   <GitChangeset at 24:6970b057cffe>
   <GitChangeset at 8:dd80b0f6cf50>
   <GitChangeset at 7:ff7ca51e58c5>

Note that ``history`` attribute is computed lazily and returned list is reversed
- changesets are retrieved from most recent to oldest.

Show me the difference!
~~~~~~~~~~~~~~~~~~~~~~~

Here we present naive implementation of diff table for the given file node
located at ``vcs/nodes.py``. First we have to get the node from repository.
After that we retrieve last changeset for which the file has been modified
and we create a html file using `difflib`_.

.. code-block:: python

   >>> new = repo.get_changeset(repo.tags['v0.1.11'])
   >>> old = repo.get_changeset(repo.tags['v0.1.10'])
   >>> f_old = old.get_node('vcs/nodes.py')
   >>> f_new = new.get_node('vcs/nodes.py')
   >>> f_old = repo.get_changeset(81).get_node(f.path)
   >>> out = open('/tmp/out.html', 'w')
   >>> from difflib import HtmlDiff
   >>> hd = HtmlDiff(tabsize=4)
   >>> diffs = hd.make_file(f_new.content.split('\n'), f_old.content.split('\n'))
   >>> out.write(diffs)
   >>> out.close()

Now open file at ``/tmp/out.html`` in your favorite browser.

.. _difflib: http://docs.python.org/library/difflib.html

.. _alternatives:

Alternatives
------------

There are a couple of alternatives to vcs:

- `anyvc <http://pypi.python.org/pypi/anyvc/>`_ actively maintained, similar to
  vcs (in a way it tries to abstract scms), supports more backends (svn, bzr);
  as far as we can tell it's main heart of Pida_; it's main focus however is on
  working directories, does not support in memory commits or history
  traversing;

- `pyvcs <https://github.com/alex/pyvcs>`_ not actively maintained; this
  package focus on history and repository traversing, does not support commits
  at all; is much simpler from vcs so may be used if you don't need full repos
  interface

.. note::
   If you know any other similar Python library, please let us know!

.. _pida: http://pida.co.uk/

===
VCS
===

.. image:: https://secure.travis-ci.org/codeinn/vcs.png?branch=master
  :target: http://travis-ci.org/codeinn/vcs

various version control system management abstraction layer for python.

------------
Introduction
------------

``vcs`` is abstraction layer over various version control systems. It is
designed as feature-rich Python_ library with clean *API*.

vcs uses `Semantic Versioning <http://semver.org/>`_

**Features**

- Common *API* for SCM backends
- Fetching repositories data lazily
- Simple caching mechanism so we don't hit repo too often
- Simple commit api
- Smart and powerfull in memory changesets
- Working directory support


-------------
Documentation
-------------

Online documentation for development version is available at
http://packages.python.org/vcs/.

You may also build documentation for yourself - go into ``docs/`` and run::

   make html

.. _python: http://www.python.org/
.. _Sphinx: http://sphinx.pocoo.org/
.. _mercurial: http://mercurial.selenic.com/
.. _git: http://git-scm.com/

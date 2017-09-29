.. _index:

Welcome to vcs's documentation!
===============================

``vcs`` is abstraction layer over various version control systems. It is
designed as feature-rich Python_ library with clear :ref:`API`.

vcs uses `Semantic Versioning <http://semver.org/>`_

**Features**

- Common :ref:`API <API>` for SCM :ref:`backends <api-backends>`
- Fetching repositories data lazily
- Simple caching mechanism so we don't hit repo too often
- In memory commits API
- Command Line Interface

**Incoming**

- Full working directories support
- Extra backends: Subversion, Bazaar

Documentation
=============

**Installation:**

.. toctree::
   :maxdepth: 1

   quickstart
   installation
   usage/index
   contribute
   alternatives
   api/index
   license

Other topics
============

* :ref:`genindex`
* :ref:`search`

.. _python: http://www.python.org/
.. _mercurial: http://mercurial.selenic.com/
.. _subversion: http://subversion.tigris.org/
.. _git: http://git-scm.com/

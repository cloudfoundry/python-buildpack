.. _installation:

Installation
============

``vcs`` is simply, pure python package. However, it makes use of various
*version control systems* and thus, would require some third part libraries
and they may have some deeper dependencies.

Requirements
------------

Below is a table which shows requirements for each backend.

+------------+---------------------+---------+---------------------+
| SCM        | Backend             | Alias   | Requirements        |
+============+=====================+=========+=====================+
| Mercurial_ | ``vcs.backend.hg``  | ``hg``  | - mercurial_ >= 1.9 |
+------------+---------------------+---------+---------------------+
| Git_       | ``vcs.backend.git`` | ``git`` | - git_ >= 1.7       |
|            |                     |         | - Dulwich_ >= 0.8   |
+------------+---------------------+---------+---------------------+

Install from Cheese Shop
------------------------

Easiest way to install ``vcs`` is to run::

   easy_install vcs

Or::

   pip install vcs

If you prefer to install manually simply grab latest release from
http://pypi.python.org/pypi/vcs, decompress archive and run::

   python setup.py install

Development
-----------

In order to test the package you'd need all backends underlying libraries (see
table above) and unittest2_ as we use it to run test suites.

Here is a full list of packages needed to run test suite:

+-----------+---------------------------------------+
| Package   | Homepage                              |
+===========+=======================================+
| mock      | http://pypi.python.org/pypi/mock      |
+-----------+---------------------------------------+
| unittest2 | http://pypi.python.org/pypi/unittest2 |
+-----------+---------------------------------------+
| mercurial | http://mercurial.selenic.com/         |
+-----------+---------------------------------------+
| git       | http://git-scm.com                    |
+-----------+---------------------------------------+
| dulwich   | http://pypi.python.org/pypi/dulwich   |
+-----------+---------------------------------------+

.. _unittest2: http://pypi.python.org/pypi/unittest2
.. _git: http://git-scm.com
.. _dulwich: http://pypi.python.org/pypi/dulwich
.. _mercurial: http://mercurial.selenic.com/

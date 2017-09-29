.. _contribute:

How to contribute
=================

There are a lot of ways people may contribute to vcs. First of all, if you spot
a bug please file it at `issue tracker <http://github.com/codeinn/vcs/issues/>`_.
Moreover, if you feel you can fix the problem on your own and want to contribute
to vcs, just fork from your preferred scm and send us your pull request.

.. note::
   Oh, some codes may be very ugly. If you spot ugly code, file a bug/clean it/
   make it more readable/send us a note.


Repositories
------------

As we do *various version control systems*, we also try to be flexible at where
code resides and therefor, could be accessed by wider audience.

- Main git repository is at https://github.com/codeinn/vcs/
- Main Mercurial repository is at https://bitbucket.org/marcinkuzminski/vcs/

We are going to create one *official* repository per supported
:ref:`backend <api-backends>`.


How to write backend
--------------------

Don't see you favorite scm at supported :ref:`backends <api-backends>` but like
vcs :ref:`API <api>`? Writing your own backend is in fact very simple process -
all the backends should extend from :ref:`base backend <api-base-backend>`,
however, as there are a few classes that needs to be written (repository,
changeset, in-memory-changeset, workingdir) one would probably want to review
existing backends' codebase.

Tests
-----

Tests are fundamental to vcs development process. In fact we try to do TDD_ as
much as we can, however it doesn't always fit well with open source projects
development.  Nevertheless, we don't accept patches without tests. So... test,
damn it! Whole heavy-lifting is done for you already, anyway (unless you don't
intend to write new backend)!


.. _TDD: http://en.wikipedia.org/wiki/Test-driven_development

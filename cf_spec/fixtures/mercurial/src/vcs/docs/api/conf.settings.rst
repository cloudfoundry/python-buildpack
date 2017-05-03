.. _api-conf:

vcs.conf.settings
=================

.. automodule:: vcs.conf.settings


.. setting:: ARCHIVE_SPECS

ARCHIVE_SPECS
-------------

Dictionary with mapping of *archive types* to *mimetypes*.

Default::

    {
        'tar': ('application/x-tar', '.tar'),
        'tbz2': ('application/x-bzip2', '.tar.bz2'),
        'tgz': ('application/x-gzip', '.tar.gz'),
        'zip': ('application/zip', '.zip'),
    }


.. setting:: BACKENDS

BACKENDS
--------

Dictionary with mapping of *scm aliases* to *backend repository classes*.

Default::

    {
        'hg': 'vcs.backends.hg.MercurialRepository',
        'git': 'vcs.backends.git.GitRepository',
    }


.. setting:: VCSRC_PATH

VCSRC_PATH
----------

Points at a path where :command:`ExecutionManager` should look for module
specified by user. By default it would be ``$HOME/.vimrc``.

This value may be modified by setting system environment ``VCSRC_PATH``
(accessible at ``os.environ['VCSRC_PATH']``).

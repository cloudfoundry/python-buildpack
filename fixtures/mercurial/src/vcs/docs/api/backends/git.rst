.. _api-backends-git:

vcs.backends.git
================

.. automodule:: vcs.backends.git

GitRepository
-------------

.. autoclass:: vcs.backends.git.GitRepository
   :members:

GitChangeset
------------

.. autoclass:: vcs.backends.git.GitChangeset
   :members:
   :inherited-members:
   :undoc-members:
   :show-inheritance:

   .. autoattribute:: id

      Returns same as ``raw_id`` attribute.

   .. autoattribute:: raw_id

      Returns raw string identifing this changeset (40-length sha)

   .. autoattribute:: short_id

      Returns shortened version of ``raw_id`` (first 12 characters)

   .. autoattribute:: revision

      Returns integer representing changeset.

   .. autoattribute:: parents

      Returns list of parents changesets.

   .. autoattribute:: added

      Returns list of added ``FileNode`` objects.

   .. autoattribute:: changed

      Returns list of changed ``FileNode`` objects.

   .. autoattribute:: removed

      Returns list of removed ``RemovedFileNode`` objects.

      .. note::
         Remember that those ``RemovedFileNode`` instances are only dummy
         ``FileNode`` objects and trying to access most of it's attributes or
         methods would raise ``NodeError`` exception.

GitInMemoryChangeset
--------------------

.. autoclass:: vcs.backends.git.GitInMemoryChangeset
   :members:
   :inherited-members:
   :undoc-members:
   :show-inheritance:

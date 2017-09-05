.. _api-backends-hg:

vcs.backends.hg
===============

.. automodule:: vcs.backends.hg

MercurialRepository
-------------------

.. autoclass:: vcs.backends.hg.MercurialRepository
   :members:

MercurialChangeset
------------------

.. autoclass:: vcs.backends.hg.MercurialChangeset
   :members:
   :inherited-members:
   :undoc-members:
   :show-inheritance:

   .. autoattribute:: id

      Returns shorter version of mercurial's changeset hexes.

   .. autoattribute:: raw_id

      Returns raw string identifying this changeset (40-length hex)

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

MercurialInMemoryChangeset
--------------------------

.. autoclass:: MercurialInMemoryChangeset
   :members:
   :inherited-members:
   :undoc-members:
   :show-inheritance:

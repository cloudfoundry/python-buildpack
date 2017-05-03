.. _usage-vcsrc:

vcsrc
=====

During commands execution, vcs tries to build a module specified at
:setting:`VCSRC_PATH`. This would fail silently if module does not exist
- user is responsible for creating own *vcsrc* file.

Creating own commands
---------------------

User may create his own commands and add them dynamically by pointing them
at ``vcs.cli.registry`` map. Here is very simple example of how *vcsrc*
file could look like::

    from vcs import cli

    class AuthorsCommand(cli.ChangesetCommand):

        def pre_process(self, repo):
            self.authors = {}

        def handle_changeset(self, changeset, **options):
            if changeset.author not in self.authors:
                self.authors[changeset.author] = 0
            self.authors[changeset.author] += 1

        def post_process(self, repo, **options):
            for author, changesets_number in self.authors.iteritems():
                message = '%s : %s' % (author, changesets_number)
                self.stdout.write(message + '\n')

    cli.registry['authors'] = AuthorsCommand

This would create ``AuthorsCommand`` that is mapped to ``authors`` subcommand.
In order to run the command user would enter repository and type into terminal::

    vcs authors

As we have subclassed :command:`ChangesetCommand`, we also got all the
changesets specified options. User may see whole help with following command::

    vcs authors -h

.. note::
   Please refer to :ref:`api-cli` for more information about the basic commands.

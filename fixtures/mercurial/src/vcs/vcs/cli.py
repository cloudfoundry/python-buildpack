"""
Command line interface for VCS
------------------------------

This module provides foundations for creating, executing and registering
terminal commands for vcs. Moreover, :command:`ExecutionManager` makes it
possible for user to create own code (at *.vcsrc* file).
"""
import os
import sys
import vcs
import copy
import errno
from optparse import OptionParser
from optparse import make_option
from vcs.conf import settings
from vcs.exceptions import CommandError
from vcs.exceptions import VCSError
from vcs.utils.fakemod import create_module
from vcs.utils.helpers import get_scm
from vcs.utils.helpers import parse_changesets
from vcs.utils.helpers import parse_datetime
from vcs.utils.imports import import_class
from vcs.utils.ordered_dict import OrderedDict
from vcs.utils.paths import abspath
from vcs.utils.progressbar import ColoredProgressBar
from vcs.utils.termcolors import colorize


COMPLETION_ENV_NAME = 'VCS_AUTO_COMPLETE'

registry = {
    'cat':         'vcs.commands.cat.CatCommand',
    'completion':  'vcs.commands.completion.CompletionCommand',
    'log':         'vcs.commands.log.LogCommand',
    'standup':     'vcs.commands.standup.StandupCommand',
    'summary':     'vcs.commands.summary.SummaryCommand',
}

class ExecutionManager(object):
    """
    Class for command execution management.
    """

    def __init__(self, argv=None, stdout=None, stderr=None):
        if argv:
            self.prog_name = argv[0]
            self.argv = argv[1:]
        else:
            self.prog_name = sys.argv[0]
            self.argv = sys.argv[1:]
        self.stdout = stdout or sys.stdout
        self.stderr = stderr or sys.stderr
        self.vimrc = self.get_vcsrc()
        self.registry = registry.copy()

    def get_vcsrc(self):
        """
        Returns in-memory created module pointing at user's configuration
        and extra code/commands. By default tries to create module from
        :setting:`VCSRC_PATH`.
        """
        try:
            vimrc = create_module('vcsrc', settings.VCSRC_PATH)
        except IOError:
            self.stderr.write("No module or package at %s\n"
                % settings.VCSRC_PATH)
            vimrc = None
        return vimrc

    def get_argv_for_command(self):
        """
        Returns stripped arguments that would be passed into the command.
        """
        argv = [a for a in self.argv]
        argv.insert(0, self.prog_name)
        return argv

    def execute(self):
        """
        Executes whole process of parsing and running command.
        """
        self.autocomplete()
        if len(self.argv):
            cmd = self.argv[0]
            cmd_argv = self.get_argv_for_command()
            self.run_command(cmd, cmd_argv)
        else:
            self.show_help()

    def autocomplete(self):
        if COMPLETION_ENV_NAME not in os.environ:
            return
        cwords = os.environ['COMP_WORDS'].split()[1:]
        cword = int(os.environ['COMP_CWORD'])
        try:
            current = cwords[cword-1]
        except IndexError:
            current = ''
        cmd_names = self.get_commands().keys()

        if current:
            self.stdout.write(unicode(' '.join(
                [name for name in cmd_names if name.startswith(current)])))

        sys.exit(1)

    def get_command_class(self, cmd):
        """
        Returns command class from the registry for a given ``cmd``.

        :param cmd: command to run (key at the registry)
        """
        try:
            cmdpath = self.registry[cmd]
        except KeyError:
            raise CommandError("No such command %r" % cmd)
        if isinstance(cmdpath, basestring):
            Command = import_class(cmdpath)
        else:
            Command = cmdpath
        return Command

    def get_commands(self):
        """
        Returns commands stored in the registry.
        """
        commands = OrderedDict()
        for cmd in sorted(self.registry.keys()):
            commands[cmd] = self.get_command_class(cmd)
        return commands

    def run_command(self, cmd, argv):
        """
        Runs command.

        :param cmd: command to run (key at the registry)
        :param argv: arguments passed to the command
        """
        try:
            Command = self.get_command_class(cmd)
        except CommandError, e:
            self.stderr.write(str(e) + '\n')
            self.show_help()
            sys.exit(-1)
        command = Command(stdout=self.stdout, stderr=self.stderr)
        command.run_from_argv(argv)

    def show_help(self):
        """
        Prints help text about available commands.
        """
        output = [
            'Usage %s subcommand [options] [args]' % self.prog_name,
            '',
            'Available commands:',
            '',
        ]
        for cmd in self.get_commands():
            output.append('  %s' % cmd)
        output += ['', '']
        self.stdout.write(u'\n'.join(output))


class BaseCommand(object):
    """
    Base command class.
    """
    help = ''
    args = ''
    option_list = (
        make_option('--debug', action='store_true', dest='debug',
            default=False, help='Enter debug mode before raising exception'),
        make_option('--traceback', action='store_true', dest='traceback',
            default=False, help='Print traceback in case of an error'),
    )

    def __init__(self, stdout=None, stderr=None):
        self.stdout = stdout or sys.stdout
        self.stderr = stderr or sys.stderr

    def get_version(self):
        """
        Returns version of vcs.
        """
        return vcs.get_version()

    def usage(self, subcommand):
        """
        Returns *how to use command* text.
        """
        usage = ' '.join(['%prog', subcommand, '[options]'])
        if self.args:
            usage = '%s %s' % (usage, str(self.args))
        return usage

    def get_option_list(self):
        """
        Returns options specified at ``self.option_list``.
        """
        return self.option_list

    def get_parser(self, prog_name, subcommand):
        """
        Returns parser for given ``prog_name`` and ``subcommand``.

        :param prog_name: vcs main script name
        :param subcommand: command name
        """
        parser = OptionParser(
            prog=prog_name,
            usage=self.usage(subcommand),
            version=self.get_version(),
            option_list=sorted(self.get_option_list()))
        return parser

    def print_help(self, prog_name, subcommand):
        """
        Prints parser's help.

        :param prog_name: vcs main script name
        :param subcommand: command name
        """
        parser = self.get_parser(prog_name, subcommand)
        parser.print_help()

    def run_from_argv(self, argv):
        """
        Runs command for given arguments.

        :param argv: arguments
        """
        parser = self.get_parser(argv[0], argv[1])
        options, args = parser.parse_args(argv[2:])
        self.execute(*args, **options.__dict__)

    def execute(self, *args, **options):
        """
        Executes whole process of parsing arguments, running command and
        trying to catch errors.
        """
        try:
            self.handle(*args, **options)
        except CommandError, e:
            if options['debug']:
                try:
                    import ipdb
                    ipdb.set_trace()
                except ImportError:
                    import pdb
                    pdb.set_trace()
            sys.stderr.write(colorize('ERROR: ', fg='red'))
            self.stderr.write('%s\n' % e)
            sys.exit(1)
        except Exception, e:
            if isinstance(e, IOError) and getattr(e, 'errno') == errno.EPIPE:
                sys.exit(0)
            if options['debug']:
                try:
                    import ipdb
                    ipdb.set_trace()
                except ImportError:
                    import pdb
                    pdb.set_trace()
            if options.get('traceback'):
                import traceback
                self.stderr.write(u'\n'.join((
                    '=========',
                    'TRACEBACK',
                    '=========', '', '',
                )))
                traceback.print_exc(file=self.stderr)
            sys.stderr.write(colorize('ERROR: ', fg='red'))
            self.stderr.write('%s\n' % e)
            sys.exit(1)

    def handle(self, *args, **options):
        """
        This method must be implemented at subclass.
        """
        raise NotImplementedError()


class RepositoryCommand(BaseCommand):
    """
    Base repository command.
    """

    def __init__(self, stdout=None, stderr=None, repo=None):
        """
        Accepts extra argument:

        :param repo: repository instance. If not given, repository would be
          calculated based on current directory.
        """
        if repo is None:
            curdir = abspath(os.curdir)
            try:
                scm, path = get_scm(curdir, search_up=True)
                self.repo = vcs.get_repo(path, scm)
            except VCSError:
                raise CommandError('Repository not found')
        else:
            self.repo = repo
        super(RepositoryCommand, self).__init__(stdout, stderr)

    def pre_process(self, repo, **options):
        """
        This method would be run at the beginning of ``handle`` method. Does
        nothing by default.
        """

    def post_process(self, repo, **options):
        """
        This method would be run at the end of ``handle`` method. Does
        nothing by default.
        """

    def handle(self, *args, **options):
        """
        Runs ``pre_process``, ``handle_repo`` and ``post_process`` methods, in
        that order.
        """
        self.pre_process(self.repo)
        self.handle_repo(self.repo, *args, **options)
        self.post_process(self.repo, **options)

    def handle_repo(self, repo, *args, **options):
        """
        Handles given repository. This method must be implemented at subclass.
        """
        raise NotImplementedError()


class ChangesetCommand(RepositoryCommand):
    """
    Subclass of :command:`RepositoryCommand`.

    **Extra options**

    * ``show_progress_bar``: specifies if bar indicating progress of processed
      changesets should be shown.
    """
    show_progress_bar = False

    option_list = RepositoryCommand.option_list + (
        make_option('--author', action='store', dest='author',
            help='Show changes committed by specified author only.'),
        make_option('-r', '--reversed', action='store_true', dest='reversed',
            default=False, help='Iterates in asceding order.'),
        make_option('-b', '--branch', action='store', dest='branch',
            help='Narrow changesets to chosen branch. If not given, '
                 'working directory branch is used. For bare repository '
                 'default would be SCM\'s default branch (i.e. master for git)'),
        make_option('--all', action='store_true', dest='all',
            default=False, help='Show changesets across all branches.'),

        make_option('--since', '--start-date', action='store', dest='start_date',
            help='Show only changesets not younger than specified '
                 'start date.'),
        make_option('--until', '--end-date', action='store', dest='end_date',
            help='Show only changesets not older than specified '
                 'end date.'),

        make_option('--limit', action='store', dest='limit', default=None,
            help='Limit number of showed changesets.'),
    )

    def show_changeset(self, changeset, **options):
        author = options.get('author')
        if author:
            if author.startswith('*') and author.endswith('*') and \
                author.strip('*') in changeset.author:
                return True
            if author.startswith('*') and changeset.author.endswith(
                author.strip('*')):
                return True
            if author.endswith('*') and changeset.author.startswith(
                author.strip('*')):
                return True
            return changeset.author == author
        return True

    def get_changesets(self, repo, **options):
        """
        Returns generator of changesets from given ``repo`` for given
        ``options``.

        :param repo: repository instance. Same as ``self.repo``.

        **Available options**

        * ``start_date``: only changesets not older than this parameter would be
          generated
        * ``end_date``: only changesets not younger than this parameter would be
          generated
        * ``start``: changeset's ID from which changesets would be generated
        * ``end``: changeset's ID to which changesets would be generated
        * ``branch``: branch for which changesets would be generated. If ``all``
          flag is specified, this option would be ignored. By default, branch
          would by tried to retrieved from working directory.
        * ``all``: return changesets from all branches
        * ``reversed``: by default changesets are returned in date order. If
          this flag is set to ``True``, reverse order would be applied.
        * ``limit``: if specified, show no more changesets than this value.
          Default is ``None``.
        """
        branch_name = None
        if not options['all']:
            branch_name = options.get('branch') or repo.workdir.get_branch()
        if options.get('start_date'):
            options['start_date'] = parse_datetime(options['start_date'])
        if options.get('end_date'):
            options['end_date'] = parse_datetime(options['end_date'])
        changesets = repo.get_changesets(
            start=options.get('start'),
            end=options.get('end', options.get('main')),
            start_date=options.get('start_date'),
            end_date=options.get('end_date'),
            branch_name=branch_name,
            reverse=not options.get('reversed', False),
        )
        try:
            limit = int(options.get('limit'))
        except (ValueError, TypeError):
            limit = None

        count = 0
        for changeset in changesets:
            if self.show_changeset(changeset, **options):
                yield changeset
                count += 1
                if count == limit:
                    break

    def handle_repo(self, repo, *args, **options):
        opts = copy.copy(options)
        if len(args) == 1:
            opts.update(parse_changesets(args[0]))
        elif len(args) > 1:
            raise CommandError("Wrong changeset ID(s) given")
        if options.get('limit') and not options['limit'].isdigit():
            raise CommandError("Limit must be a number")
        changesets = self.get_changesets(repo, **opts)
        self.iter_changesets(repo, changesets, **options)

    def iter_changesets(self, repo, changesets, **options):
        changesets = list(changesets)
        if self.show_progress_bar:
            progressbar = self.get_progressbar(len(changesets), **options)
            progressbar.render(0)
        else:
            progressbar = None
        i = 0
        for changeset in changesets:
            self.handle_changeset(changeset, **options)
            i += 1
            if progressbar:
                progressbar.render(i)

    def get_progressbar(self, total, **options):
        """
        Returns progress bar instance for a given ``total`` number of clicks
        it should do.
        """
        progressbar = ColoredProgressBar(total)
        progressbar.steps_label = 'Commit'
        progressbar.elements += ['eta', 'time']
        return progressbar

    def handle_changeset(self, changeset, **options):
        """
        Handles single changeset. Must be implemented at subclass.
        """
        raise NotImplementedError()


class SingleChangesetCommand(RepositoryCommand):
    """
    Single changeset command. Convenient if command has to operate on single
    changeset rather than whole generator. For usage i.e. with command that
    handles node(s) from single changeset.

    **Extra options**

    * ``min_args``: minimal number of arguements to parse. Default is ``1``.
    """

    min_args = 1

    option_list = RepositoryCommand.option_list + (
        make_option('-c', '--commit', action='store', dest='changeset_id',
            default=None, help='Use specific commit. By default we use HEAD/tip'),
    )

    def get_changeset(self, **options):
        """
        Returns changeset for given ``options``.
        """
        cid = options.get('changeset_id', None)
        return self.repo.get_changeset(cid)

    def handle_repo(self, repo, *args, **options):
        if len(args) < self.min_args:
            raise CommandError("At least %s arguments required" % self.min_args)
        changeset = self.get_changeset(**options)
        for arg in args:
            self.handle_arg(changeset, arg, **options)

    def handle_arg(self, changeset, arg, **options):
        """
        Handles single argument for chosen ``changeset``. Must be implemented at
        subclass.

        :param changeset: chosen (by ``--commit`` option) changeset
        :param arg: single argument from arguments list
        """
        raise NotImplementedError()

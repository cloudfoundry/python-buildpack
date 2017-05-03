from __future__ import with_statement

import datetime
import mock
import sys
import subprocess
import vcs
import vcs.cli
from contextlib import nested
from StringIO import StringIO
from vcs.cli import BaseCommand
from vcs.cli import ExecutionManager
from vcs.cli import make_option
from vcs.nodes import FileNode
from vcs.tests.base import BackendTestMixin
from vcs.tests.conf import SCM_TESTS
from vcs.utils.compat import unittest


class DummyExecutionManager(ExecutionManager):

    def get_vcsrc(self):
        return None


class TestExecutionManager(unittest.TestCase):

    def test_default_argv(self):
        with mock.patch.object(sys, 'argv', ['vcs', 'foo', 'bar']):
            manager = DummyExecutionManager()
            self.assertEqual(manager.argv, ['foo', 'bar'])

    def test_default_prog_name(self):
        with mock.patch.object(sys, 'argv', ['vcs', 'foo', 'bar']):
            manager = DummyExecutionManager()
            self.assertEqual(manager.prog_name, 'vcs')

    def test_default_stdout(self):
        stream = StringIO()
        with mock.patch.object(sys, 'stdout', stream):
            manager = DummyExecutionManager()
            self.assertEqual(manager.stdout, stream)

    def test_default_stderr(self):
        stream = StringIO()
        with mock.patch.object(sys, 'stderr', stream):
            manager = DummyExecutionManager()
            self.assertEqual(manager.stderr, stream)

    def test_get_vcsrc(self):
        with nested(mock.patch('vcs.conf.settings.VCSRC_PATH', 'foobar'),
            mock.patch('vcs.cli.create_module')) as (VP, m):
            # Use not-dummy manager here as we need to test get_vcsrc behavior
            m.return_value = mock.Mock()
            manager = ExecutionManager()
            self.assertEqual(manager.vimrc, m.return_value)
            m.assert_called_once_with('vcsrc', 'foobar')

    def test_get_command_class(self):
        with mock.patch.object(vcs.cli, 'registry', {
            'foo': 'decimal.Decimal',
            'bar': 'socket.socket'}):
            manager = DummyExecutionManager()
            from decimal import Decimal
            from socket import socket
            self.assertEqual(manager.get_command_class('foo'), Decimal)
            self.assertEqual(manager.get_command_class('bar'), socket)

    def test_get_commands(self):
        with mock.patch.object(vcs.cli, 'registry', {
            'mock': mock,
            'foo': 'vcs.cli.BaseCommand',
            'bar': 'vcs.tests.test_cli.DummyExecutionManager'}):
            manager = DummyExecutionManager()

            self.assertItemsEqual(manager.get_commands(), {
                'foo': BaseCommand,
                'bar': DummyExecutionManager,
                'mock': mock,
            })

    def test_run_command(self):
        manager = DummyExecutionManager(stdout=StringIO(),
            stderr=StringIO())

        class Command(BaseCommand):

            def run_from_argv(self, argv):
                self.stdout.write(u'foo')
                self.stderr.write(u'bar')

        with mock.patch.object(manager, 'get_command_class') as m:
            m.return_value = Command
            manager.run_command('cmd', manager.argv)
            self.assertEqual(manager.stdout.getvalue(), u'foo')
            self.assertEqual(manager.stderr.getvalue(), u'bar')

    def test_execute_calls_run_command_if_argv_given(self):
        manager = DummyExecutionManager(argv=['vcs', 'show', '-h'])
        manager.run_command = mock.Mock()
        manager.execute()
        # we also check argv passed to the command
        manager.run_command.assert_called_once_with('show',
            ['vcs', 'show', '-h'])

    def test_execute_calls_show_help_if_argv_not_given(self):
        manager = DummyExecutionManager(argv=['vcs'])
        manager.show_help = mock.Mock()
        manager.execute()
        manager.show_help.assert_called_once_with()

    def test_show_help_writes_to_stdout(self):
        manager = DummyExecutionManager(stdout=StringIO(), stderr=StringIO())
        manager.show_help()
        self.assertGreater(len(manager.stdout.getvalue()), 0)


class TestBaseCommand(unittest.TestCase):

    def test_default_stdout(self):
        stream = StringIO()
        with mock.patch.object(sys, 'stdout', stream):
            command = BaseCommand()
            command.stdout.write(u'foobar')
            self.assertEqual(sys.stdout.getvalue(), u'foobar')

    def test_default_stderr(self):
        stream = StringIO()
        with mock.patch.object(sys, 'stderr', stream):
            command = BaseCommand()
            command.stderr.write(u'foobar')
            self.assertEqual(sys.stderr.getvalue(), u'foobar')

    def test_get_version(self):
        command = BaseCommand()
        self.assertEqual(command.get_version(), vcs.get_version())

    def test_usage(self):
        command = BaseCommand()
        command.args = 'foo'
        self.assertEqual(command.usage('bar'),
            '%prog bar [options] foo')

    def test_get_parser(self):

        class Command(BaseCommand):
            option_list = (
                make_option('-f', '--foo', action='store', dest='foo',
                    default='bar'),
            )
        command = Command()
        parser = command.get_parser('vcs', 'cmd')
        options, args = parser.parse_args(['-f', 'FOOBAR', 'arg1', 'arg2'])
        self.assertEqual(options.__dict__['foo'], 'FOOBAR')
        self.assertEqual(args, ['arg1', 'arg2'])

    def test_execute(self):
        command = BaseCommand()
        command.handle = mock.Mock()
        args = ['bar']
        kwargs = {'debug': True}
        command.execute(*args, **kwargs)
        command.handle.assert_called_once_with(*args, **kwargs)

    def test_run_from_argv(self):
        command = BaseCommand()
        argv = ['vcs', 'foo', '--debug', 'bar', '--traceback']
        command.handle = mock.Mock()
        command.run_from_argv(argv)
        args = ['bar']
        kwargs = {'debug': True, 'traceback': True}
        command.handle.assert_called_once_with(*args, **kwargs)


class RealCliMixin(BackendTestMixin):

    @classmethod
    def _get_commits(cls):
        commits = [
            {
                'message': u'Initial commit',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [FileNode('file1', content='Foobar')],
            },
            {
                'message': u'Added a file',
                'author': u'Joe Doe <joe.doe@example.com>',
                'date': datetime.datetime(2010, 1, 1, 20),
                'added': [FileNode('file2', content='Foobar')],
            },
        ]
        return commits

    def test_log_command(self):
        cmd = 'vcs log --template "\$message"'
        process = subprocess.Popen(cmd, cwd=self.repo.path, shell=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        so, se = process.communicate()
        self.assertEqual(process.returncode, 0)
        self.assertEqual(so.splitlines(), [
            'Added a file',
            'Initial commit',
        ])


# For each backend create test case class
for alias in SCM_TESTS:
    attrs = {
        'backend_alias': alias,
    }
    cls_name = ''.join(('%s real cli' % alias).title().split())
    bases = (RealCliMixin, unittest.TestCase)
    globals()[cls_name] = type(cls_name, bases, attrs)


if __name__ == '__main__':
    unittest.main()

"""
Utilities for tests only. These are not or should not be used normally -
functions here are crafted as we don't want to use ``vcs`` to verify tests.
"""
import os
import re
import sys

from subprocess import Popen


class VCSTestError(Exception):
    pass


def run_command(cmd, args):
    """
    Runs command on the system with given ``args``.
    """
    command = ' '.join((cmd, args))
    p = Popen(command, shell=True)
    status = os.waitpid(p.pid, 0)[1]
    return status


def eprint(msg):
    """
    Prints given ``msg`` into sys.stderr as nose test runner hides all output
    from sys.stdout by default and if we want to pipe stream somewhere we don't
    need those verbose messages anyway.
    Appends line break.
    """
    sys.stderr.write(msg)
    sys.stderr.write('\n')


class SCMFetcher(object):

    def __init__(self, alias, test_repo_path, remote_repo, clone_cmd):
        """
        :param clone_cmd: command which would clone remote repository; pass
          only first bits - remote path and destination would be appended
          using ``remote_repo`` and ``test_repo_path``
        """
        self.alias = alias
        self.test_repo_path = test_repo_path
        self.remote_repo = remote_repo
        self.clone_cmd = clone_cmd

    def setup(self):
        if not os.path.isdir(self.test_repo_path):
            self.fetch_repo()

    def fetch_repo(self):
        """
        Tries to fetch repository from remote path.
        """
        remote = self.remote_repo
        eprint("Fetching repository %s into %s" % (remote, self.test_repo_path))
        run_command(self.clone_cmd,  '%s %s' % (remote, self.test_repo_path))


def get_normalized_path(path):
    """
    If given path exists, new path would be generated and returned. Otherwise
    same whats given is returned. Assumes that there would be no more than
    10000 same named files.
    """
    if os.path.exists(path):
        dir, basename = os.path.split(path)
        splitted_name = basename.split('.')
        if len(splitted_name) > 1:
            ext = splitted_name[-1]
        else:
            ext = None
        name = '.'.join(splitted_name[:-1])
        matcher = re.compile(r'^.*-(\d{5})$')
        start = 0
        m = matcher.match(name)
        if not m:
            # Haven't append number yet so return first
            newname = '%s-00000' % name
            newpath = os.path.join(dir, newname)
            if ext:
                newpath = '.'.join((newpath, ext))
            return get_normalized_path(newpath)
        else:
            start = int(m.group(1)[-5:]) + 1
            for x in xrange(start, 10000):
                newname = name[:-5] + str(x).rjust(5, '0')
                newpath = os.path.join(dir, newname)
                if ext:
                    newpath = '.'.join((newpath, ext))
                if not os.path.exists(newpath):
                    return newpath
        raise VCSTestError("Couldn't compute new path for %s" % path)
    return path

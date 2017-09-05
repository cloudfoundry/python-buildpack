import datetime
import errno

from vcs.backends.base import BaseInMemoryChangeset
from vcs.exceptions import RepositoryError

from vcs.utils.hgcompat import memfilectx, memctx, hex, tolocal


class MercurialInMemoryChangeset(BaseInMemoryChangeset):

    def commit(self, message, author, parents=None, branch=None, date=None,
            **kwargs):
        """
        Performs in-memory commit (doesn't check workdir in any way) and
        returns newly created ``Changeset``. Updates repository's
        ``revisions``.

        :param message: message of the commit
        :param author: full username, i.e. "Joe Doe <joe.doe@example.com>"
        :param parents: single parent or sequence of parents from which commit
          would be derieved
        :param date: ``datetime.datetime`` instance. Defaults to
          ``datetime.datetime.now()``.
        :param branch: branch name, as string. If none given, default backend's
          branch would be used.

        :raises ``CommitError``: if any error occurs while committing
        """
        self.check_integrity(parents)

        from .repository import MercurialRepository
        if not isinstance(message, unicode) or not isinstance(author, unicode):
            raise RepositoryError('Given message and author needs to be '
                                  'an <unicode> instance got %r & %r instead'
                                  % (type(message), type(author)))

        if branch is None:
            branch = MercurialRepository.DEFAULT_BRANCH_NAME
        kwargs['branch'] = branch

        def filectxfn(_repo, memctx, path):
            """
            Marks given path as added/changed/removed in a given _repo. This is
            for internal mercurial commit function.
            """

            # check if this path is removed
            if path in (node.path for node in self.removed):
                # Raising exception is a way to mark node for removal
                raise IOError(errno.ENOENT, '%s is deleted' % path)

            # check if this path is added
            for node in self.added:
                if node.path == path:
                    return memfilectx(path=node.path,
                        data=(node.content.encode('utf8')
                              if not node.is_binary else node.content),
                        islink=False,
                        isexec=node.is_executable,
                        copied=False)

            # or changed
            for node in self.changed:
                if node.path == path:
                    return memfilectx(path=node.path,
                        data=(node.content.encode('utf8')
                              if not node.is_binary else node.content),
                        islink=False,
                        isexec=node.is_executable,
                        copied=False)

            raise RepositoryError("Given path haven't been marked as added,"
                                  "changed or removed (%s)" % path)

        parents = [None, None]
        for i, parent in enumerate(self.parents):
            if parent is not None:
                parents[i] = parent._ctx.node()

        if date and isinstance(date, datetime.datetime):
            date = date.ctime()

        commit_ctx = memctx(repo=self.repository._repo,
            parents=parents,
            text='',
            files=self.get_paths(),
            filectxfn=filectxfn,
            user=author,
            date=date,
            extra=kwargs)

        loc = lambda u: tolocal(u.encode('utf-8'))

        # injecting given _repo params
        commit_ctx._text = loc(message)
        commit_ctx._user = loc(author)
        commit_ctx._date = date

        # TODO: Catch exceptions!
        n = self.repository._repo.commitctx(commit_ctx)
        # Returns mercurial node
        self._commit_ctx = commit_ctx  # For reference
        # Update vcs repository object & recreate mercurial _repo
        # new_ctx = self.repository._repo[node]
        # new_tip = self.repository.get_changeset(new_ctx.hex())
        new_id = hex(n)
        self.repository.revisions.append(new_id)
        self._repo = self.repository._get_repo(create=False)
        self.repository.branches = self.repository._get_branches()
        tip = self.repository.get_changeset()
        self.reset()
        return tip

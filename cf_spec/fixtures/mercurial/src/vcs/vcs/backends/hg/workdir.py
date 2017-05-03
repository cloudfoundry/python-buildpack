from vcs.backends.base import BaseWorkdir
from vcs.exceptions import BranchDoesNotExistError

from vcs.utils.hgcompat import hg_merge


class MercurialWorkdir(BaseWorkdir):

    def get_branch(self):
        return self.repository._repo.dirstate.branch()

    def get_changeset(self):
        wk_dir_id = self.repository._repo[None].parents()[0].hex()
        return self.repository.get_changeset(wk_dir_id)

    def checkout_branch(self, branch=None):
        if branch is None:
            branch = self.repository.DEFAULT_BRANCH_NAME
        if branch not in self.repository.branches:
            raise BranchDoesNotExistError

        hg_merge.update(self.repository._repo, branch, False, False, None)

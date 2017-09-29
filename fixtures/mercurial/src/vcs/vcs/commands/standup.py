from vcs.commands.log import LogCommand


class StandupCommand(LogCommand):

    def handle_repo(self, repo, **options):
        options['all'] = True
        options['start_date'] = '1day'
        username = repo.get_user_name()
        options['author'] = username + '*'
        return super(StandupCommand, self).handle_repo(repo, **options)

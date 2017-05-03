from vcs.cli import make_option
from vcs.cli import ChangesetCommand
from vcs.utils.filesize import filesizeformat


class SummaryCommand(ChangesetCommand):
    show_progress_bar = True

    option_list = ChangesetCommand.option_list + (
        make_option('-s', '--with-changesets-size', action='store_true',
            dest='changeset_size', default=False,
            help='Counts size of filenodes from each commit [may be *heavy*]'),
    )

    def __init__(self, *args, **kwargs):
        super(SummaryCommand, self).__init__(*args, **kwargs)
        self.total_size = 0
        self.authors = {}
        self.start_date = None
        self.last_date = None

    def handle_changeset(self, changeset, **options):
        if options['changeset_size']:
            self.total_size += changeset.size

        if changeset.author not in self.authors:
            self.authors[changeset.author] = {
                'changeset_id_list': [changeset.raw_id],
            }
        else:
            self.authors[changeset.author]['changeset_id_list'].append(
                changeset.raw_id)

        if not self.start_date or changeset.date < self.start_date:
            self.start_date = changeset.date
        if not self.last_date or changeset.date > self.last_date:
            self.last_date = changeset.date



    def post_process(self, repo, **options):
        stats = [
            ('Total repository size [HDD]', filesizeformat(repo.size)),
            ('Total number of commits', len(repo)),
            ('Total number of branches', len(repo.branches)),
            ('Total number of tags', len(repo.tags)),
            ('Total number of authors', len(self.authors)),
            ('Avarage number of commits/author',
                float(len(repo)) / len(self.authors)),
            ('Avarage number of commits/day',
                float(len(repo)) / (self.last_date - self.start_date).days),
        ]
        if options['changeset_size']:
            stats.append(('Total size in all changesets',
                filesizeformat(self.total_size)))

        max_label_size = max(len(label) for label, value in stats)
        output = ['']
        output.extend([
            '%s: %s' % (label.rjust(max_label_size+3), value)
            for label, value in stats
        ])
        output.append('')
        output.append('')
        self.stdout.write(u'\n'.join(output))

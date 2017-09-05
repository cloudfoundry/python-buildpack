from vcs.cli import BaseCommand
from vcs.cli import COMPLETION_ENV_NAME


COMPLETION_TEMPLATE = '''
# %(prog_name)s bash completion start
_%(prog_name)s_completion()
{
    COMPREPLY=( $( COMP_WORDS="${COMP_WORDS[*]}" \\
                   COMP_CWORD=$COMP_CWORD \\
                   %(ENV_VAR_NAME)s=1 $1 ) )
}
complete -o default -F _%(prog_name)s_completion %(prog_name)s
# %(prog_name)s bash completion end

'''


class CompletionCommand(BaseCommand):
    help = ''.join((
        'Prints out shell snippet that once evaluated would allow '
        'this command utility to use completion abilities.',
    ))
    template = COMPLETION_TEMPLATE

    def get_completion_snippet(self):
        return self.template % {'prog_name': 'vcs',
            'ENV_VAR_NAME': COMPLETION_ENV_NAME}

    def handle(self, **options):
        self.stdout.write(self.get_completion_snippet())

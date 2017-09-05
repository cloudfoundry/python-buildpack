# encoding: UTF-8
import sys
import datetime
import string

from vcs.utils.filesize import filesizeformat
from vcs.utils.helpers import get_total_seconds


class ProgressBarError(Exception):
    pass

class AlreadyFinishedError(ProgressBarError):
    pass


class ProgressBar(object):

    default_elements = ['percentage', 'bar', 'steps']

    def __init__(self, steps=100, stream=None, elements=None):
        self.step = 0
        self.steps = steps
        self.stream = stream or sys.stderr
        self.bar_char = '='
        self.width = 50
        self.separator = ' | '
        self.elements = elements or self.default_elements
        self.started = None
        self.finished = False
        self.steps_label = 'Step'
        self.time_label = 'Time'
        self.eta_label = 'ETA'
        self.speed_label = 'Speed'
        self.transfer_label = 'Transfer'

    def __str__(self):
        return self.get_line()

    def __iter__(self):
        start = self.step
        end = self.steps + 1
        for x in xrange(start, end):
            self.render(x)
            yield x

    def get_separator(self):
        return self.separator

    def get_bar_char(self):
        return self.bar_char

    def get_bar(self):
        char = self.get_bar_char()
        perc = self.get_percentage()
        length = int(self.width * perc / 100)
        bar = char * length
        bar = bar.ljust(self.width)
        return bar

    def get_elements(self):
        return self.elements

    def get_template(self):
        separator = self.get_separator()
        elements = self.get_elements()
        return string.Template(separator.join((('$%s' % e) for e in elements)))

    def get_total_time(self, current_time=None):
        if current_time is None:
            current_time = datetime.datetime.now()
        if not self.started:
            return datetime.timedelta()
        return current_time - self.started

    def get_rendered_total_time(self):
        delta = self.get_total_time()
        if not delta:
            ttime = '-'
        else:
            ttime = str(delta)
        return '%s %s' % (self.time_label, ttime)

    def get_eta(self, current_time=None):
        if current_time is None:
            current_time = datetime.datetime.now()
        if self.step == 0:
            return datetime.timedelta()
        total_seconds = get_total_seconds(self.get_total_time())
        eta_seconds = total_seconds * self.steps / self.step - total_seconds
        return datetime.timedelta(seconds=int(eta_seconds))

    def get_rendered_eta(self):
        eta = self.get_eta()
        if not eta:
            eta = '--:--:--'
        else:
            eta = str(eta).rjust(8)
        return '%s: %s' % (self.eta_label, eta)

    def get_percentage(self):
        return float(self.step) / self.steps * 100

    def get_rendered_percentage(self):
        perc = self.get_percentage()
        return ('%s%%' % (int(perc))).rjust(5)

    def get_rendered_steps(self):
        return '%s: %s/%s' % (self.steps_label, self.step, self.steps)

    def get_rendered_speed(self, step=None, total_seconds=None):
        if step is None:
            step = self.step
        if total_seconds is None:
            total_seconds = get_total_seconds(self.get_total_time())
        if step <= 0 or total_seconds <= 0:
            speed = '-'
        else:
            speed = filesizeformat(float(step) / total_seconds)
        return '%s: %s/s' % (self.speed_label, speed)

    def get_rendered_transfer(self, step=None, steps=None):
        if step is None:
            step = self.step
        if steps is None:
            steps = self.steps

        if steps <= 0:
            return '%s: -' % self.transfer_label
        total = filesizeformat(float(steps))
        if step <= 0:
            transferred = '-'
        else:
            transferred = filesizeformat(float(step))
        return '%s: %s / %s' % (self.transfer_label, transferred, total)

    def get_context(self):
        return {
            'percentage': self.get_rendered_percentage(),
            'bar': self.get_bar(),
            'steps': self.get_rendered_steps(),
            'time': self.get_rendered_total_time(),
            'eta': self.get_rendered_eta(),
            'speed': self.get_rendered_speed(),
            'transfer': self.get_rendered_transfer(),
        }

    def get_line(self):
        template = self.get_template()
        context = self.get_context()
        return template.safe_substitute(**context)

    def write(self, data):
        self.stream.write(data)

    def render(self, step):
        if not self.started:
            self.started = datetime.datetime.now()
        if self.finished:
            raise AlreadyFinishedError
        self.step = step
        self.write('\r%s' % self)
        if step == self.steps:
            self.finished = True
        if step == self.steps:
            self.write('\n')


"""
termcolors.py

Grabbed from Django (http://www.djangoproject.com)
"""

color_names = ('black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white')
foreground = dict([(color_names[x], '3%s' % x) for x in range(8)])
background = dict([(color_names[x], '4%s' % x) for x in range(8)])

RESET = '0'
opt_dict = {'bold': '1', 'underscore': '4', 'blink': '5', 'reverse': '7', 'conceal': '8'}

def colorize(text='', opts=(), **kwargs):
    """
    Returns your text, enclosed in ANSI graphics codes.

    Depends on the keyword arguments 'fg' and 'bg', and the contents of
    the opts tuple/list.

    Returns the RESET code if no parameters are given.

    Valid colors:
        'black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white'

    Valid options:
        'bold'
        'underscore'
        'blink'
        'reverse'
        'conceal'
        'noreset' - string will not be auto-terminated with the RESET code

    Examples:
        colorize('hello', fg='red', bg='blue', opts=('blink',))
        colorize()
        colorize('goodbye', opts=('underscore',))
        print colorize('first line', fg='red', opts=('noreset',))
        print 'this should be red too'
        print colorize('and so should this')
        print 'this should not be red'
    """
    code_list = []
    if text == '' and len(opts) == 1 and opts[0] == 'reset':
        return '\x1b[%sm' % RESET
    for k, v in kwargs.iteritems():
        if k == 'fg':
            code_list.append(foreground[v])
        elif k == 'bg':
            code_list.append(background[v])
    for o in opts:
        if o in opt_dict:
            code_list.append(opt_dict[o])
    if 'noreset' not in opts:
        text = text + '\x1b[%sm' % RESET
    return ('\x1b[%sm' % ';'.join(code_list)) + text

def make_style(opts=(), **kwargs):
    """
    Returns a function with default parameters for colorize()

    Example:
        bold_red = make_style(opts=('bold',), fg='red')
        print bold_red('hello')
        KEYWORD = make_style(fg='yellow')
        COMMENT = make_style(fg='blue', opts=('bold',))
    """
    return lambda text: colorize(text, opts, **kwargs)

NOCOLOR_PALETTE = 'nocolor'
DARK_PALETTE = 'dark'
LIGHT_PALETTE = 'light'

PALETTES = {
    NOCOLOR_PALETTE: {
        'ERROR':        {},
        'NOTICE':       {},
        'SQL_FIELD':    {},
        'SQL_COLTYPE':  {},
        'SQL_KEYWORD':  {},
        'SQL_TABLE':    {},
        'HTTP_INFO':         {},
        'HTTP_SUCCESS':      {},
        'HTTP_REDIRECT':     {},
        'HTTP_NOT_MODIFIED': {},
        'HTTP_BAD_REQUEST':  {},
        'HTTP_NOT_FOUND':    {},
        'HTTP_SERVER_ERROR': {},
    },
    DARK_PALETTE: {
        'ERROR':        { 'fg': 'red', 'opts': ('bold',) },
        'NOTICE':       { 'fg': 'red' },
        'SQL_FIELD':    { 'fg': 'green', 'opts': ('bold',) },
        'SQL_COLTYPE':  { 'fg': 'green' },
        'SQL_KEYWORD':  { 'fg': 'yellow' },
        'SQL_TABLE':    { 'opts': ('bold',) },
        'HTTP_INFO':         { 'opts': ('bold',) },
        'HTTP_SUCCESS':      { },
        'HTTP_REDIRECT':     { 'fg': 'green' },
        'HTTP_NOT_MODIFIED': { 'fg': 'cyan' },
        'HTTP_BAD_REQUEST':  { 'fg': 'red', 'opts': ('bold',) },
        'HTTP_NOT_FOUND':    { 'fg': 'yellow' },
        'HTTP_SERVER_ERROR': { 'fg': 'magenta', 'opts': ('bold',) },
    },
    LIGHT_PALETTE: {
        'ERROR':        { 'fg': 'red', 'opts': ('bold',) },
        'NOTICE':       { 'fg': 'red' },
        'SQL_FIELD':    { 'fg': 'green', 'opts': ('bold',) },
        'SQL_COLTYPE':  { 'fg': 'green' },
        'SQL_KEYWORD':  { 'fg': 'blue' },
        'SQL_TABLE':    { 'opts': ('bold',) },
        'HTTP_INFO':         { 'opts': ('bold',) },
        'HTTP_SUCCESS':      { },
        'HTTP_REDIRECT':     { 'fg': 'green', 'opts': ('bold',) },
        'HTTP_NOT_MODIFIED': { 'fg': 'green' },
        'HTTP_BAD_REQUEST':  { 'fg': 'red', 'opts': ('bold',) },
        'HTTP_NOT_FOUND':    { 'fg': 'red' },
        'HTTP_SERVER_ERROR': { 'fg': 'magenta', 'opts': ('bold',) },
    }
}
DEFAULT_PALETTE = DARK_PALETTE

# ---------------------------- #
# --- End of termcolors.py --- #
# ---------------------------- #


class ColoredProgressBar(ProgressBar):

    BAR_COLORS = (
        (10, 'red'),
        (30, 'magenta'),
        (50, 'yellow'),
        (99, 'green'),
        (100, 'blue'),
    )

    def get_line(self):
        line = super(ColoredProgressBar, self).get_line()
        perc = self.get_percentage()
        if perc > 100:
            color = 'blue'
        for max_perc, color in self.BAR_COLORS:
            if perc <= max_perc:
                break
        return colorize(line, fg=color)


class AnimatedProgressBar(ProgressBar):

    def get_bar_char(self):
        chars = '-/|\\'
        if self.step >= self.steps:
            return '='
        return chars[self.step % len(chars)]


class BarOnlyProgressBar(ProgressBar):

    default_elements = ['bar', 'steps']

    def get_bar(self):
        bar = super(BarOnlyProgressBar, self).get_bar()
        perc = self.get_percentage()
        perc_text = '%s%%' % int(perc)
        text = (' %s%% ' % (perc_text)).center(self.width, '=')
        L = text.find(' ')
        R = text.rfind(' ')
        bar = ' '.join((bar[:L], perc_text, bar[R:]))
        return bar


class AnimatedColoredProgressBar(AnimatedProgressBar,
                                 ColoredProgressBar):
    pass


class BarOnlyColoredProgressBar(ColoredProgressBar,
                                BarOnlyProgressBar):
    pass



def main():
    import time

    print "Standard progress bar..."
    bar = ProgressBar(30)
    for x in xrange(1, 31):
            bar.render(x)
            time.sleep(0.02)
    bar.stream.write('\n')
    print

    print "Empty bar..."
    bar = ProgressBar(50)
    bar.render(0)
    print
    print

    print "Colored bar..."
    bar = ColoredProgressBar(20)
    for x in bar:
        time.sleep(0.01)
    print

    print "Animated char bar..."
    bar = AnimatedProgressBar(20)
    for x in bar:
        time.sleep(0.01)
    print

    print "Animated + colored char bar..."
    bar = AnimatedColoredProgressBar(20)
    for x in bar:
        time.sleep(0.01)
    print

    print "Bar only ..."
    bar = BarOnlyProgressBar(20)
    for x in bar:
        time.sleep(0.01)
    print

    print "Colored, longer bar-only, eta, total time ..."
    bar = BarOnlyColoredProgressBar(40)
    bar.width = 60
    bar.elements += ['time', 'eta']
    for x in bar:
        time.sleep(0.01)
    print
    print

    print "File transfer bar, breaks after 2 seconds ..."
    total_bytes = 1024 * 1024 * 2
    bar = ProgressBar(total_bytes)
    bar.width = 50
    bar.elements.remove('steps')
    bar.elements += ['transfer', 'time', 'eta', 'speed']
    for x in xrange(0, bar.steps, 1024):
        bar.render(x)
        time.sleep(0.01)
        now = datetime.datetime.now()
        if now - bar.started >= datetime.timedelta(seconds=2):
            break
    print
    print



if __name__ == '__main__':
    main()

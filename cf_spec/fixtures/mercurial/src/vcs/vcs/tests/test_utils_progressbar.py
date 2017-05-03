from __future__ import with_statement

import sys
import datetime
from StringIO import StringIO
from vcs.utils.helpers import get_total_seconds
from vcs.utils.progressbar import AlreadyFinishedError
from vcs.utils.progressbar import ProgressBar
from vcs.utils.compat import unittest


class TestProgressBar(unittest.TestCase):

    def test_default_get_separator(self):
        bar = ProgressBar()
        bar.separator = '\t'
        self.assertEquals(bar.get_separator(), '\t')

    def test_cast_to_str(self):
        bar = ProgressBar()
        self.assertEquals(str(bar), bar.get_line())

    def test_default_get_bar_char(self):
        bar = ProgressBar()
        bar.bar_char = '#'
        self.assertEquals(bar.get_bar_char(), '#')

    def test_default_get_elements(self):
        bar = ProgressBar(elements=['foo', 'bar'])
        self.assertItemsEqual(bar.get_elements(), ['foo', 'bar'])

    def test_get_template(self):
        bar = ProgressBar()
        bar.elements = ['foo', 'bar']
        bar.separator = ' '
        self.assertEquals(bar.get_template().template, '$foo $bar')

    def test_default_stream_is_sys_stderr(self):
        bar = ProgressBar()
        self.assertEquals(bar.stream, sys.stderr)

    def test_get_percentage(self):
        bar = ProgressBar()
        bar.steps = 120
        bar.step = 60
        self.assertEquals(bar.get_percentage(), 50.0)
        bar.steps = 100
        bar.step = 9
        self.assertEquals(bar.get_percentage(), 9.0)

    def test_get_rendered_percentage(self):
        bar = ProgressBar()
        bar.steps = 100
        bar.step = 10.5
        self.assertEquals(bar.get_percentage(), 10.5)

    def test_bar_width(self):
        bar = ProgressBar()
        bar.width = 30
        self.assertEquals(len(bar.get_bar()), 30)

    def test_write(self):
        stream = StringIO()
        bar = ProgressBar()
        bar.stream = stream
        bar.write('foobar')
        self.assertEquals(stream.getvalue(), 'foobar')

    def test_change_stream(self):
        stream1 = StringIO()
        stream2 = StringIO()
        bar = ProgressBar()
        bar.stream = stream1
        bar.write('foo')
        bar.stream = stream2
        bar.write('bar')
        self.assertEquals(stream2.getvalue(), 'bar')

    def test_render_writes_new_line_at_last_step(self):
        bar = ProgressBar()
        bar.stream = StringIO()
        bar.steps = 5
        bar.render(5)
        self.assertEquals(bar.stream.getvalue()[-1], '\n')

    def test_initial_step_is_zero(self):
        bar = ProgressBar()
        self.assertEquals(bar.step, 0)

    def test_iter_starts_from_current_step(self):
        bar = ProgressBar()
        bar.stream = StringIO()
        bar.steps = 20
        bar.step = 5
        stepped = list(bar)
        self.assertEquals(stepped[0], 5)

    def test_iter_ends_at_last_step(self):
        bar = ProgressBar()
        bar.stream = StringIO()
        bar.steps = 20
        bar.step = 5
        stepped = list(bar)
        self.assertEquals(stepped[-1], 20)

    def test_get_total_time(self):
        bar = ProgressBar()
        now = datetime.datetime.now()
        bar.started = now - datetime.timedelta(days=1)
        self.assertEqual(bar.get_total_time(now), datetime.timedelta(days=1))

    def test_get_total_time_returns_empty_timedelta_if_not_yet_started(self):
        bar = ProgressBar()
        self.assertEquals(bar.get_total_time(), datetime.timedelta())

    def test_get_render_total_time(self):
        p = ProgressBar()
        p.time_label = 'FOOBAR'
        self.assertTrue(p.get_rendered_total_time().startswith('FOOBAR'))

    def test_get_eta(self):
        bar = ProgressBar(100)
        bar.stream = StringIO()

        bar.render(50)
        now = datetime.datetime.now()
        delta = now - bar.started
        self.assertEquals(get_total_seconds(bar.get_eta(now)),
            int(get_total_seconds(delta) * 0.5))

        bar.render(75)
        now = datetime.datetime.now()
        delta = now - bar.started
        self.assertEquals(get_total_seconds(bar.get_eta(now)),
            int(get_total_seconds(delta) * 0.25))

    def test_get_rendered_eta(self):
        bar = ProgressBar(100)
        bar.eta_label = 'foobar'
        self.assertTrue(bar.get_rendered_eta().startswith('foobar'))

    def test_get_rendered_steps(self):
        bar = ProgressBar(100)
        bar.steps_label = 'foobar'
        self.assertTrue(bar.get_rendered_steps().startswith('foobar'))

    def test_get_rendered_speed_respects_speed_label(self):
        bar = ProgressBar(100)
        bar.speed_label = 'foobar'
        self.assertTrue(bar.get_rendered_speed().startswith('foobar'))

    def test_get_rendered_speed(self):
        B = 1
        KB = B * 1024
        MB = KB * 1024
        GB = MB * 1024

        bar = ProgressBar(KB)
        self.assertEqual(bar.get_rendered_speed(512, 1), 'Speed: 512 B/s')
        self.assertEqual(bar.get_rendered_speed(512, 2), 'Speed: 256 B/s')
        self.assertEqual(bar.get_rendered_speed(900, 3), 'Speed: 300 B/s')

        bar = ProgressBar(GB * 10)
        self.assertEqual(bar.get_rendered_speed(KB, 1), 'Speed: 1 KB/s')
        self.assertEqual(bar.get_rendered_speed(MB, 1), 'Speed: 1.0 MB/s')
        self.assertEqual(bar.get_rendered_speed(GB * 4, 2), 'Speed: 2.00 GB/s')
        self.assertEqual(bar.get_rendered_speed(GB * 5, 2), 'Speed: 2.50 GB/s')

    def test_get_rendered_transfer_respects_transfer_label(self):
        bar = ProgressBar(100)
        bar.transfer_label = 'foobar'
        self.assertTrue(bar.get_rendered_transfer(0).startswith('foobar'))
        self.assertTrue(bar.get_rendered_transfer(10).startswith('foobar'))

    def test_get_rendered_transfer(self):
        B = 1
        KB = B * 1024
        MB = KB * 1024
        GB = MB * 1024

        bar = ProgressBar()
        self.assertEqual(bar.get_rendered_transfer(12, 100),
            'Transfer: 12 B / 100 B')
        self.assertEqual(bar.get_rendered_transfer(KB * 5, MB),
            'Transfer: 5 KB / 1.0 MB')
        self.assertEqual(bar.get_rendered_transfer(GB * 2.3, GB * 10),
            'Transfer: 2.30 GB / 10.00 GB')


    def test_context(self):
        bar = ProgressBar()
        context = bar.get_context()
        self.assertItemsEqual(context, [
            'bar',
            'percentage',
            'time',
            'eta',
            'steps',
            'speed',
            'transfer',
        ])

    def test_context_has_correct_bar(self):
        bar = ProgressBar()
        context = bar.get_context()
        self.assertEquals(context['bar'], bar.get_bar())

    def test_context_has_correct_percentage(self):
        bar = ProgressBar(100)
        bar.step = 50
        percentage = bar.get_context()['percentage']
        self.assertEquals(percentage, bar.get_rendered_percentage())

    def test_context_has_correct_total_time(self):
        bar = ProgressBar(100)
        time = bar.get_context()['time']
        self.assertEquals(time, bar.get_rendered_total_time())

    def test_context_has_correct_eta(self):
        bar = ProgressBar(100)
        eta = bar.get_context()['eta']
        self.assertEquals(eta, bar.get_rendered_eta())

    def test_context_has_correct_steps(self):
        bar = ProgressBar(100)
        steps = bar.get_context()['steps']
        self.assertEquals(steps, bar.get_rendered_steps())

    def context_has_correct_speed(self):
        bar = ProgressBar(100)
        speed = bar.get_context()['speed']
        self.assertEquals(speed, bar.get_rendered_speed())

    def test_render_raises_error_if_bar_already_finished(self):
        bar = ProgressBar(10)
        bar.stream = StringIO()
        bar.render(10)

        with self.assertRaises(AlreadyFinishedError):
            bar.render(0)


if __name__ == '__main__':
    unittest.main()

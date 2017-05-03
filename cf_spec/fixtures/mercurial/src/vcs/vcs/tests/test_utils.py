from __future__ import with_statement

import os
import mock
import time
import shutil
import tempfile
import datetime
from vcs.utils.compat import unittest
from vcs.utils.paths import get_dirs_for_path
from vcs.utils.helpers import get_dict_for_attrs
from vcs.utils.helpers import get_scm
from vcs.utils.helpers import get_scms_for_path
from vcs.utils.helpers import get_total_seconds
from vcs.utils.helpers import parse_changesets
from vcs.utils.helpers import parse_datetime
from vcs.utils import author_email, author_name
from vcs.utils.paths import get_user_home
from vcs.exceptions import VCSError

from vcs.tests.conf import TEST_HG_REPO, TEST_GIT_REPO, TEST_TMP_PATH


class PathsTest(unittest.TestCase):

    def _test_get_dirs_for_path(self, path, expected):
        """
        Tests if get_dirs_for_path returns same as expected.
        """
        expected = sorted(expected)
        result = sorted(get_dirs_for_path(path))
        self.assertEqual(result, expected,
            msg="%s != %s which was expected result for path %s"
            % (result, expected, path))

    def test_get_dirs_for_path(self):
        path = 'foo/bar/baz/file'
        paths_and_results = (
            ('foo/bar/baz/file', ['foo', 'foo/bar', 'foo/bar/baz']),
            ('foo/bar/', ['foo', 'foo/bar']),
            ('foo/bar', ['foo']),
        )
        for path, expected in paths_and_results:
            self._test_get_dirs_for_path(path, expected)


    def test_get_scm(self):
        self.assertEqual(('hg', TEST_HG_REPO), get_scm(TEST_HG_REPO))
        self.assertEqual(('git', TEST_GIT_REPO), get_scm(TEST_GIT_REPO))

    def test_get_two_scms_for_path(self):
        multialias_repo_path = os.path.join(TEST_TMP_PATH, 'hg-git-repo-2')
        if os.path.isdir(multialias_repo_path):
            shutil.rmtree(multialias_repo_path)

        os.mkdir(multialias_repo_path)

        self.assertRaises(VCSError, get_scm, multialias_repo_path)

    def test_get_scm_error_path(self):
        self.assertRaises(VCSError, get_scm, 'err')

    def test_get_scms_for_path(self):
        dirpath = tempfile.gettempdir()
        new = os.path.join(dirpath, 'vcs-scms-for-path-%s' % time.time())
        os.mkdir(new)
        self.assertEqual(get_scms_for_path(new), [])

        os.mkdir(os.path.join(new, '.tux'))
        self.assertEqual(get_scms_for_path(new), [])

        os.mkdir(os.path.join(new, '.git'))
        self.assertEqual(set(get_scms_for_path(new)), set(['git']))

        os.mkdir(os.path.join(new, '.hg'))
        self.assertEqual(set(get_scms_for_path(new)), set(['git', 'hg']))


class TestParseChangesets(unittest.TestCase):

    def test_main_is_returned_correctly(self):
        self.assertEqual(parse_changesets('123456'), {
            'start': None,
            'main': '123456',
            'end': None,
        })

    def test_start_is_returned_correctly(self):
        self.assertEqual(parse_changesets('aaabbb..'), {
            'start': 'aaabbb',
            'main': None,
            'end': None,
        })

    def test_end_is_returned_correctly(self):
        self.assertEqual(parse_changesets('..cccddd'), {
            'start': None,
            'main': None,
            'end': 'cccddd',
        })

    def test_that_two_or_three_dots_are_allowed(self):
        text1 = 'a..b'
        text2 = 'a...b'
        self.assertEqual(parse_changesets(text1), parse_changesets(text2))

    def test_that_input_is_stripped_first(self):
        text1 = 'a..bb'
        text2 = '  a..bb\t\n\t '
        self.assertEqual(parse_changesets(text1), parse_changesets(text2))

    def test_that_exception_is_raised(self):
        text = '123456.789012' # single dot is not recognized
        with self.assertRaises(ValueError):
            parse_changesets(text)

    def test_non_alphanumeric_raises_exception(self):
        with self.assertRaises(ValueError):
            parse_changesets('aaa@bbb')


class TestParseDatetime(unittest.TestCase):

    def test_datetime_text(self):
        self.assertEqual(parse_datetime('2010-04-07 21:29:41'),
            datetime.datetime(2010, 4, 7, 21, 29, 41))

    def test_no_seconds(self):
        self.assertEqual(parse_datetime('2010-04-07 21:29'),
            datetime.datetime(2010, 4, 7, 21, 29))

    def test_date_only(self):
        self.assertEqual(parse_datetime('2010-04-07'),
            datetime.datetime(2010, 4, 7))

    def test_another_format(self):
        self.assertEqual(parse_datetime('04/07/10 21:29:41'),
            datetime.datetime(2010, 4, 7, 21, 29, 41))

    def test_now(self):
        self.assertTrue(parse_datetime('now') - datetime.datetime.now() <
            datetime.timedelta(seconds=1))

    def test_today(self):
        today = datetime.date.today()
        self.assertEqual(parse_datetime('today'),
            datetime.datetime(*today.timetuple()[:3]))

    def test_yesterday(self):
        yesterday = datetime.date.today() - datetime.timedelta(days=1)
        self.assertEqual(parse_datetime('yesterday'),
            datetime.datetime(*yesterday.timetuple()[:3]))

    def test_tomorrow(self):
        tomorrow = datetime.date.today() + datetime.timedelta(days=1)
        args = tomorrow.timetuple()[:3] + (23, 59, 59)
        self.assertEqual(parse_datetime('tomorrow'), datetime.datetime(*args))

    def test_days(self):
        timestamp = datetime.datetime.today() - datetime.timedelta(days=3)
        args = timestamp.timetuple()[:3] + (0, 0, 0, 0)
        expected = datetime.datetime(*args)
        self.assertEqual(parse_datetime('3d'), expected)
        self.assertEqual(parse_datetime('3 d'), expected)
        self.assertEqual(parse_datetime('3 day'), expected)
        self.assertEqual(parse_datetime('3 days'), expected)

    def test_weeks(self):
        timestamp = datetime.datetime.today() - datetime.timedelta(days=3 * 7)
        args = timestamp.timetuple()[:3] + (0, 0, 0, 0)
        expected = datetime.datetime(*args)
        self.assertEqual(parse_datetime('3w'), expected)
        self.assertEqual(parse_datetime('3 w'), expected)
        self.assertEqual(parse_datetime('3 week'), expected)
        self.assertEqual(parse_datetime('3 weeks'), expected)

    def test_mixed(self):
        timestamp = datetime.datetime.today() - datetime.timedelta(days=2 * 7 + 3)
        args = timestamp.timetuple()[:3] + (0, 0, 0, 0)
        expected = datetime.datetime(*args)
        self.assertEqual(parse_datetime('2w3d'), expected)
        self.assertEqual(parse_datetime('2w 3d'), expected)
        self.assertEqual(parse_datetime('2w 3 days'), expected)
        self.assertEqual(parse_datetime('2 weeks 3 days'), expected)


class TestAuthorExtractors(unittest.TestCase):
    TEST_AUTHORS = [('Marcin Kuzminski <marcin@python-works.com>',
                    ('Marcin Kuzminski', 'marcin@python-works.com')),
                  ('Marcin Kuzminski Spaces < marcin@python-works.com >',
                    ('Marcin Kuzminski Spaces', 'marcin@python-works.com')),
                  ('Marcin Kuzminski <marcin.kuzminski@python-works.com>',
                    ('Marcin Kuzminski', 'marcin.kuzminski@python-works.com')),
                  ('mrf RFC_SPEC <marcin+kuzminski@python-works.com>',
                    ('mrf RFC_SPEC', 'marcin+kuzminski@python-works.com')),
                  ('username <user@email.com>',
                    ('username', 'user@email.com')),
                  ('username <user@email.com',
                   ('username', 'user@email.com')),
                  ('broken missing@email.com',
                   ('broken', 'missing@email.com')),
                  ('<justemail@mail.com>',
                   ('', 'justemail@mail.com')),
                  ('justname',
                   ('justname', '')),
                  ('Mr Double Name withemail@email.com ',
                   ('Mr Double Name', 'withemail@email.com')),
                  ]

    def test_author_email(self):

        for test_str, result in self.TEST_AUTHORS:
            self.assertEqual(result[1], author_email(test_str))


    def test_author_name(self):

        for test_str, result in self.TEST_AUTHORS:
            self.assertEqual(result[0], author_name(test_str))


class TestGetDictForAttrs(unittest.TestCase):

    def test_returned_dict_has_expected_attrs(self):
        obj = mock.Mock()
        obj.NOT_INCLUDED = 'this key/value should not be included'
        obj.CONST = True
        obj.foo = 'aaa'
        obj.attrs = {'foo': 'bar'}
        obj.date = datetime.datetime(2010, 12, 31)
        obj.count = 1001

        self.assertEqual(get_dict_for_attrs(obj, ['CONST', 'foo', 'attrs',
            'date', 'count']), {
            'CONST': True,
            'foo': 'aaa',
            'attrs': {'foo': 'bar'},
            'date': datetime.datetime(2010, 12, 31),
            'count': 1001,
        })


class TestGetTotalSeconds(unittest.TestCase):

    def assertTotalSecondsEqual(self, timedelta, expected_seconds):
        result = get_total_seconds(timedelta)
        self.assertEqual(result, expected_seconds,
            "We computed %s seconds for %s but expected %s"
            % (result, timedelta, expected_seconds))

    def test_get_total_seconds_returns_proper_value(self):
        self.assertTotalSecondsEqual(datetime.timedelta(seconds=1001), 1001)

    def test_get_total_seconds_returns_proper_value_for_partial_seconds(self):
        self.assertTotalSecondsEqual(datetime.timedelta(seconds=50.65), 50.65)


class TestGetUserHome(unittest.TestCase):

    @mock.patch.object(os, 'environ', {})
    def test_defaults_to_none(self):
        self.assertEqual(get_user_home(), '')

    @mock.patch.object(os, 'environ', {'HOME': '/home/foobar'})
    def test_unix_like(self):
        self.assertEqual(get_user_home(), '/home/foobar')

    @mock.patch.object(os, 'environ', {'USERPROFILE': '/Users/foobar'})
    def test_windows_like(self):
        self.assertEqual(get_user_home(), '/Users/foobar')

    @mock.patch.object(os, 'environ', {'HOME': '/home/foobar',
        'USERPROFILE': '/Users/foobar'})
    def test_prefers_home_over_userprofile(self):
        self.assertEqual(get_user_home(), '/home/foobar')


if __name__ == '__main__':
    unittest.main()

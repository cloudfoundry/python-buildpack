"""
This module provides some useful tools for ``vcs`` like annotate/diff html
output. It also includes some internal helpers.
"""
import sys
import time
import datetime


def makedate():
    lt = time.localtime()
    if lt[8] == 1 and time.daylight:
        tz = time.altzone
    else:
        tz = time.timezone
    return time.mktime(lt), tz


def aslist(obj, sep=None, strip=True):
    """
    Returns given string separated by sep as list

    :param obj:
    :param sep:
    :param strip:
    """
    if isinstance(obj, (basestring)):
        lst = obj.split(sep)
        if strip:
            lst = [v.strip() for v in lst]
        return lst
    elif isinstance(obj, (list, tuple)):
        return obj
    elif obj is None:
        return []
    else:
        return [obj]


def date_fromtimestamp(unixts, tzoffset=0):
    """
    Makes a local datetime object out of unix timestamp

    :param unixts:
    :param tzoffset:
    """

    return datetime.datetime.fromtimestamp(float(unixts))


def safe_int(val, default=None):
    """
    Returns int() of val if val is not convertable to int use default
    instead

    :param val:
    :param default:
    """

    try:
        val = int(val)
    except (ValueError, TypeError):
        val = default

    return val


def safe_unicode(str_, from_encoding=None):
    """
    safe unicode function. Does few trick to turn str_ into unicode

    In case of UnicodeDecode error we try to return it with encoding detected
    by chardet library if it fails fallback to unicode with errors replaced

    :param str_: string to decode
    :rtype: unicode
    :returns: unicode object
    """
    if isinstance(str_, unicode):
        return str_

    if not from_encoding:
        from vcs.conf import settings
        from_encoding = settings.DEFAULT_ENCODINGS

    if not isinstance(from_encoding, (list, tuple)):
        from_encoding = [from_encoding]

    try:
        return unicode(str_)
    except UnicodeDecodeError:
        pass

    for enc in from_encoding:
        try:
            return unicode(str_, enc)
        except UnicodeDecodeError:
            pass

    try:
        import chardet
        encoding = chardet.detect(str_)['encoding']
        if encoding is None:
            raise Exception()
        return str_.decode(encoding)
    except (ImportError, UnicodeDecodeError, Exception):
        return unicode(str_, from_encoding[0], 'replace')


def safe_str(unicode_, to_encoding=None):
    """
    safe str function. Does few trick to turn unicode_ into string

    In case of UnicodeEncodeError we try to return it with encoding detected
    by chardet library if it fails fallback to string with errors replaced

    :param unicode_: unicode to encode
    :rtype: str
    :returns: str object
    """

    # if it's not basestr cast to str
    if not isinstance(unicode_, basestring):
        return str(unicode_)

    if isinstance(unicode_, str):
        return unicode_

    if not to_encoding:
        from vcs.conf import settings
        to_encoding = settings.DEFAULT_ENCODINGS

    if not isinstance(to_encoding, (list, tuple)):
        to_encoding = [to_encoding]

    for enc in to_encoding:
        try:
            return unicode_.encode(enc)
        except UnicodeEncodeError:
            pass

    try:
        import chardet
        encoding = chardet.detect(unicode_)['encoding']
        if encoding is None:
            raise UnicodeEncodeError()

        return unicode_.encode(encoding)
    except (ImportError, UnicodeEncodeError):
        return unicode_.encode(to_encoding[0], 'replace')

    return safe_str


def author_email(author):
    """
    returns email address of given author.
    If any of <,> sign are found, it fallbacks to regex findall()
    and returns first found result or empty string

    Regex taken from http://www.regular-expressions.info/email.html
    """
    import re
    r = author.find('>')
    l = author.find('<')

    if l == -1 or r == -1:
        # fallback to regex match of email out of a string
        email_re = re.compile(r"""[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!"""
                              r"""#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z"""
                              r"""0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]"""
                              r"""*[a-z0-9])?""", re.IGNORECASE)
        m = re.findall(email_re, author)
        return m[0] if m else ''

    return author[l + 1:r].strip()


def author_name(author):
    """
    get name of author, or else username.
    It'll try to find an email in the author string and just cut it off
    to get the username
    """

    if not '@' in author:
        return author
    else:
        return author.replace(author_email(author), '').replace('<', '')\
            .replace('>', '').strip()

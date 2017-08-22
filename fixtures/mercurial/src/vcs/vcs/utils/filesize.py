def filesizeformat(bytes, sep=' '):
    """
    Formats the value like a 'human-readable' file size (i.e. 13 KB, 4.1 MB,
    102 B, 2.3 GB etc).

    Grabbed from Django (http://www.djangoproject.com), slightly modified.

    :param bytes: size in bytes (as integer)
    :param sep: string separator between number and abbreviation
    """
    try:
        bytes = float(bytes)
    except (TypeError, ValueError, UnicodeDecodeError):
        return '0%sB' % sep

    if bytes < 1024:
        size = bytes
        template = '%.0f%sB'
    elif bytes < 1024 * 1024:
        size = bytes / 1024
        template = '%.0f%sKB'
    elif bytes < 1024 * 1024 * 1024:
        size = bytes / 1024 / 1024
        template = '%.1f%sMB'
    else:
        size = bytes / 1024 / 1024 / 1024
        template = '%.2f%sGB'
    return template % (size, sep)

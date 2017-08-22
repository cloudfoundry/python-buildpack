from vcs.exceptions import VCSError


def import_class(class_path):
    """
    Returns class from the given path.

    For example, in order to get class located at
    ``vcs.backends.hg.MercurialRepository``:

        try:
            hgrepo = import_class('vcs.backends.hg.MercurialRepository')
        except VCSError:
            # hadle error
    """
    splitted = class_path.split('.')
    mod_path = '.'.join(splitted[:-1])
    class_name = splitted[-1]
    try:
        class_mod = __import__(mod_path, {}, {}, [class_name])
    except ImportError, err:
        msg = "There was problem while trying to import backend class. "\
            "Original error was:\n%s" % err
        raise VCSError(msg)
    cls = getattr(class_mod, class_name)

    return cls

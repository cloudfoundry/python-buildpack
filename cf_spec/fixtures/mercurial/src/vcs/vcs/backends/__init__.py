# -*- coding: utf-8 -*-
"""
    vcs.backends
    ~~~~~~~~~~~~

    Main package for scm backends

    :created_on: Apr 8, 2010
    :copyright: (c) 2010-2011 by Marcin Kuzminski, Lukasz Balcerzak.
"""
import os
from pprint import pformat
from vcs.conf import settings
from vcs.exceptions import VCSError
from vcs.utils.helpers import get_scm
from vcs.utils.paths import abspath
from vcs.utils.imports import import_class


def get_repo(path=None, alias=None, create=False):
    """
    Returns ``Repository`` object of type linked with given ``alias`` at
    the specified ``path``. If ``alias`` is not given it will try to guess it
    using get_scm method
    """
    if create:
        if not (path or alias):
            raise TypeError("If create is specified, we need path and scm type")
        return get_backend(alias)(path, create=True)
    if path is None:
        path = abspath(os.path.curdir)
    try:
        scm, path = get_scm(path, search_up=True)
        path = abspath(path)
        alias = scm
    except VCSError:
        raise VCSError("No scm found at %s" % path)
    if alias is None:
        alias = get_scm(path)[0]

    backend = get_backend(alias)
    repo = backend(path, create=create)
    return repo


def get_backend(alias):
    """
    Returns ``Repository`` class identified by the given alias or raises
    VCSError if alias is not recognized or backend class cannot be imported.
    """
    if alias not in settings.BACKENDS:
        raise VCSError("Given alias '%s' is not recognized! Allowed aliases:\n"
            "%s" % (alias, pformat(settings.BACKENDS.keys())))
    backend_path = settings.BACKENDS[alias]
    klass = import_class(backend_path)
    return klass


def get_supported_backends():
    """
    Returns list of aliases of supported backends.
    """
    return settings.BACKENDS.keys()

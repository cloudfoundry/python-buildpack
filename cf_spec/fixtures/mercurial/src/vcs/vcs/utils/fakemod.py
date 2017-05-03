import imp


def create_module(name, path):
    """
    Returns module created *on the fly*. Returned module would have name same
    as given ``name`` and would contain code read from file at the given
    ``path`` (it may also be a zip or package containing *__main__* module).
    """
    module = imp.new_module(name)
    module.__file__ = path
    execfile(path, module.__dict__)
    return module

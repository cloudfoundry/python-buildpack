class _Missing(object):

    def __repr__(self):
        return 'no value'

    def __reduce__(self):
        return '_missing'

_missing = _Missing()


class LazyProperty(object):
    """
    Decorator for easier creation of ``property`` from potentially expensive to
    calculate attribute of the class.

    Usage::

      class Foo(object):
          @LazyProperty
          def bar(self):
              print 'Calculating self._bar'
              return 42

    Taken from http://blog.pythonisito.com/2008/08/lazy-descriptors.html and
    used widely.
    """

    def __init__(self, func):
        self._func = func
        self.__module__ = func.__module__
        self.__name__ = func.__name__
        self.__doc__ = func.__doc__

    def __get__(self, obj, klass=None):
        if obj is None:
            return self
        value = obj.__dict__.get(self.__name__, _missing)
        if value is _missing:
            value = self._func(obj)
            obj.__dict__[self.__name__] = value
        return value

import threading


class ThreadLocalLazyProperty(LazyProperty):
    """
    Same as above but uses thread local dict for cache storage.
    """

    def __get__(self, obj, klass=None):
        if obj is None:
            return self
        if not hasattr(obj, '__tl_dict__'):
            obj.__tl_dict__ = threading.local().__dict__

        value = obj.__tl_dict__.get(self.__name__, _missing)
        if value is _missing:
            value = self._func(obj)
            obj.__tl_dict__[self.__name__] = value
        return value

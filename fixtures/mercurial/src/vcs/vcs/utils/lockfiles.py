import os


class LockFile(object):
    """Provides methods to obtain, check for, and release a file based lock which
    should be used to handle concurrent access to the same file.

    As we are a utility class to be derived from, we only use protected methods.

    Locks will automatically be released on destruction"""
    __slots__ = ("_file_path", "_owns_lock")

    def __init__(self, file_path):
        self._file_path = file_path
        self._owns_lock = False

    def __del__(self):
        self._release_lock()

    def _lock_file_path(self):
        """:return: Path to lockfile"""
        return "%s.lock" % (self._file_path)

    def _has_lock(self):
        """:return: True if we have a lock and if the lockfile still exists
        :raise AssertionError: if our lock-file does not exist"""
        if not self._owns_lock:
            return False

        return True

    def _obtain_lock_or_raise(self):
        """Create a lock file as flag for other instances, mark our instance as lock-holder

        :raise IOError: if a lock was already present or a lock file could not be written"""
        if self._has_lock():
            return
        lock_file = self._lock_file_path()
        if os.path.isfile(lock_file):
            raise IOError("Lock for file %r did already exist, delete %r in case the lock is illegal" % (self._file_path, lock_file))

        try:
            fd = os.open(lock_file, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0)
            os.close(fd)
        except OSError,e:
            raise IOError(str(e))

        self._owns_lock = True

    def _obtain_lock(self):
        """The default implementation will raise if a lock cannot be obtained.
        Subclasses may override this method to provide a different implementation"""
        return self._obtain_lock_or_raise()

    def _release_lock(self):
        """Release our lock if we have one"""
        if not self._has_lock():
            return

        # if someone removed our file beforhand, lets just flag this issue
        # instead of failing, to make it more usable.
        lfp = self._lock_file_path()
        try:
            # on bloody windows, the file needs write permissions to be removable.
            # Why ...
            if os.name == 'nt':
                os.chmod(lfp, 0777)
            # END handle win32
            os.remove(lfp)
        except OSError:
            pass
        self._owns_lock = False

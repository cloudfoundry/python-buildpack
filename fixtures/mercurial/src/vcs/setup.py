import os
import sys
from setuptools import setup, find_packages

vcs = __import__('vcs')
readme_file = os.path.abspath(os.path.join(os.path.dirname(__file__),
    'README.rst'))

try:
    long_description = open(readme_file).read()
except IOError, err:
    sys.stderr.write("[ERROR] Cannot find file specified as "
        "long_description (%s)\n" % readme_file)
    sys.exit(1)

install_requires = ['Pygments', 'mock']
if sys.version_info < (2, 7):
    install_requires.append('unittest2')
tests_require = install_requires + ['dulwich', 'mercurial']

setup(
    name='vcs',
    version=vcs.get_version(),
    url='https://github.com/codeinn/vcs',
    author='Marcin Kuzminski, Lukasz Balcerzak',
    author_email='marcin@python-works.com',
    description=vcs.__doc__,
    long_description=long_description,
    zip_safe=False,
    packages=find_packages(),
    scripts=[],
    install_requires=install_requires,
    tests_require=tests_require,
    test_suite='vcs.tests.collector',
    include_package_data=True,
    classifiers=[
        'Development Status :: 4 - Beta',
        'License :: OSI Approved :: MIT License',
        'Intended Audience :: Developers',
        'Programming Language :: Python',
        'Operating System :: OS Independent',
    ],
    entry_points={
        'console_scripts': [
            'vcs = vcs:main',
        ],
    },
)

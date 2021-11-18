from flask import Flask
import pytest
import os
import importlib
import sys
import traceback

MODULE_NAMES = ['numpy']
modules = {}

for m in MODULE_NAMES:
    try:
        modules[m] = importlib.import_module(m)
    except ImportError:
        modules[m] = None

app = Flask(__name__)


@app.route('/<module_name>')
def in_module_tests(module_name):
    if module_name not in modules:
        return "This module is not listed"
    try:
        result = modules[module_name].test()
        num_failures = result.failures
        result_string = "{}: number of failures={}".format(module_name, len(num_failures))
    except (NameError, ImportError, AttributeError):
        result_string = "{}: Error running test!".format(module_name)
    return result_string


@app.route('/all')
def run_all():
    results = "<br>\n".join([in_module_tests(m) for m in MODULE_NAMES])
    return str(results)


def module_version(module_name):
    m = modules[module_name]
    if m is None:
        version_string = "{}: unable to import".format(module_name)
    else:
        version_string = "{}: {}".format(module_name, m.__version__)
    return version_string


@app.route('/')
def root():
    versions = "<br>\n".join([module_version(m) for m in MODULE_NAMES])
    python_version = "\npython-version%s\n" % sys.version
    r = """<br><br>
    Imports Successful!<br>

    To test each module go to /numpy
    or test all at /all.<br>
    Test suites can take up to 10 minutes to run, main output is in app logs."""
    return python_version + versions + r

if __name__ == '__main__':
    try:
        port = int(os.getenv("PORT", 8080))
        app.run(host='0.0.0.0', port=port, debug=True)
    except Exception as e:
        traceback.print_exc()
        raise e

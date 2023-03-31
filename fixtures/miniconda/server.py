import importlib
import os
import sys

from flask import Flask

app = Flask(__name__)

port = int(os.getenv('VCAP_APP_PORT', 8080))

MODULE_NAMES = ['gunicorn']
modules = {}

for m in MODULE_NAMES:
    modules[m] = importlib.import_module(m)


def module_version(module_name):
    m = modules[module_name]
    if m is None:
        version_string = "{}: unable to import".format(module_name)
    else:
        version_string = "{}: {}".format(module_name, m.__version__)
    return version_string


@app.route('/')
def root():
    versions = "<br>" + ("<br>".join([module_version(m) for m in MODULE_NAMES]))
    python_version = "python-version%s" % sys.version
    r = "<br><br>Imports Successful!<br>"
    return python_version + versions + r


@app.route("/")
def hello():
    return "Hello, World!"


app.debug = True

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=port)

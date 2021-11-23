from flask import Flask
import os
import importlib
import sys
import traceback

MODULE_NAMES = ['numpy']
modules = {}

for m in MODULE_NAMES:
    modules[m] = importlib.import_module(m)

app = Flask(__name__)


def module_version(module_name):
    m = modules[module_name]
    if m is None:
        version_string = "{}: unable to import".format(module_name)
    else:
        version_string = "{}: {}".format(module_name, m.__version__)
    return version_string


@app.route('/')
def root():
    versions = "<br>"+("<br>".join([module_version(m) for m in MODULE_NAMES]))
    python_version = "python-version%s" % sys.version
    r = "<br><br>Imports Successful!<br>"
    return python_version + versions + r

if __name__ == '__main__':
    try:
        port = int(os.getenv("PORT", 8080))
        app.run(host='0.0.0.0', port=port, debug=True)
    except Exception as e:
        print("*** CRASHED!!!")
        traceback.print_exc()
        raise e

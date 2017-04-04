from flask import Flask, request
import subprocess
import sys

app = Flask(__name__)

@app.route("/")
def hello():
    return "max unicode: %d" % sys.maxunicode

@app.route('/execute', methods=['POST'])
def execute():
    with open('runtime.py', 'w') as f:
        f.write(request.values.get('code'))
    return subprocess.check_output(["python", "runtime.py"])

app.debug=True

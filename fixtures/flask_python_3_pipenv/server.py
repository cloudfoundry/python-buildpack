from flask import Flask, request
import subprocess
import gunicorn


app = Flask(__name__)

@app.route("/")
def hello():
    return "Hello, World with pipenv!"

@app.route('/execute', methods=['POST'])
def execute():
    with open('runtime.py', 'w') as f:
        f.write(request.values.get('code'))
    return subprocess.check_output(["python", "runtime.py"])

@app.route('/versions')
def versions():
    version = gunicorn.__version__
    return "Gunicorn version: " + version

app.debug=True

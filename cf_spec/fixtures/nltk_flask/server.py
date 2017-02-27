from flask import Flask, request
from nltk.corpus import brown
import subprocess

app = Flask(__name__)

@app.route("/")
def hello():
    return ' '.join(brown.words())

def nltktest():
    return ' '.join(brown.words())

@app.route('/execute', methods=['POST'])
def execute():
    with open('runtime.py', 'w') as f:
        f.write(request.values.get('code'))
    return subprocess.check_output(["python", "runtime.py"])

app.debug=True

from flask import Flask, request
import subprocess
import boto3

app = Flask(__name__)

@app.route("/")

def hello():
    s3_client = boto3.client('s3')
    print(s3_client)
    return "Hello, World!"

@app.route('/execute', methods=['POST'])
def execute():
    with open('runtime.py', 'w') as f:
        f.write(request.values.get('code'))
    return subprocess.check_output(["python", "runtime.py"])

app.debug=True

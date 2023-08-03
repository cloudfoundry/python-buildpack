"""Cloud Foundry test"""
import os
from flask import Flask, request

app = Flask(__name__)

port = int(os.getenv('VCAP_APP_PORT', 8080))


@app.route('/health')
def health():
    return "OK"


@app.route('/')
def base():
    return "OK"


@app.route('/add', methods=['GET'])
def add():
    a = request.args.get('a')
    b = request.args.get('b')
    try:
        a = int(a)
        b = int(b)
    except ValueError:
        return "Error: Please provide valid integer values for the 'a' and 'b' parameters."

    return str(a + b)

app.debug = True

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=port)

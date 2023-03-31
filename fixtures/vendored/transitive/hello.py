import os

from flask import Flask, jsonify

app = Flask(__name__)

port = int(os.getenv('VCAP_APP_PORT', 8080))


@app.route('/')
def welcome():
    return 'Welcome to Python on Cloud Foundry!'


@app.route('/health')
def health():
    return 'UP'


@app.route('/v1/api', methods=['GET'])
def get():
    return jsonify({'message': 'hello world'})


app.debug = True

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=port)

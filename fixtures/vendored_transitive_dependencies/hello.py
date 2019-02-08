from flask import Flask, jsonify, abort, request
import os
import sys

app = Flask(__name__)

@app.route('/')
def welcome():
    return 'Welcome to Python on Cloud Foundry!'

@app.route('/health')
def health():
    return 'UP'

@app.route('/v1/api', methods=['GET'])
def get():
    return jsonify({ 'message': 'hello world' })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=int(os.getenv('PORT', 5001)))

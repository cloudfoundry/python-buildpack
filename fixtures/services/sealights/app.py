"""Cloud Foundry test"""
from flask import Flask, request
 import logging
 from service import calc

 app = Flask(__name__)
 log = logging.getLogger(__name__)


 @app.route('/health', methods=['GET'])
 def health():
     return "OK"


 @app.route('/', methods=['GET'])
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

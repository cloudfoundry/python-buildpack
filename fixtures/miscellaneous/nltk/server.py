import os

from flask import Flask
from nltk.corpus import brown

app = Flask(__name__)

port = int(os.getenv('VCAP_APP_PORT', 8080))


@app.route("/")
def hello():
    return ' '.join(brown.sents()[0])


app.debug = True

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=port)

import os

from flask import Flask

app = Flask(__name__)

port = int(os.getenv('VCAP_APP_PORT', 8080))


@app.route("/")
def hello():
    return "Hello, World!"


@app.route("/dotnet")
def dotnet():
    command = 'dotnet --version'
    output = os.popen(command).read()
    return "dotnet: " + output


app.debug = True

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=port)

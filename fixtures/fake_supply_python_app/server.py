from flask import Flask, request
import subprocess

app = Flask(__name__)

@app.route("/")
def dotnet_version():
    return "dotnet: " + subprocess.check_output(["dotnet", "--version"]).decode("utf-8")

app.debug=True

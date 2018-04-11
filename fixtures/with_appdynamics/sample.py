"""Cloud Foundry test"""
from flask import Flask, jsonify
import os
import collections

app = Flask(__name__)

port = int(os.getenv('VCAP_APP_PORT', 8080))


@app.route('/vcap')
def vcap():
    vcap_services = os.getenv('VCAP_SERVICES', "")
    return vcap_services


@app.route('/appd')
def appd():
    env_vars = ["APPD_ACCOUNT_ACCESS_KEY", "APPD_ACCOUNT_NAME", "APPD_APP_NAME", "APPD_CONTROLLER_HOST",
                "APPD_CONTROLLER_PORT", "APPD_NODE_NAME", "APPD_SSL_ENABLED","APPD_TIER_NAME"]
    env_vars.sort()
    env_dict = collections.OrderedDict([(envKey, os.getenv(envKey))for envKey in env_vars])
    return jsonify(env_dict)


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=port)

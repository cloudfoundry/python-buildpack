from flask import Flask, request, jsonify, make_response
import os

broker = Flask(__name__)

fake_service = {
    "services": [
        {
            "id": "1e3d32a0-c979-11e7-8e98-7bebc67e38ac",
            "name": "appdynamics",
            "description": "fake appdynamics broker",
            "bindable": True,
            "tags": [],
            "metadata": {
                "displayName": "appdynamics",
                "imageUrl": "http://via.placeholder.com/100x100",
                "longDescription": "fake appdynamics broker",
                "providerDisplayName": "appdynamics",
                "documentationUrl": "http://example.com",
                "supportUrl": "http://example.com"
            },
            "plans": [
                {
                    "id": "24ba3a06-c979-11e7-a5c2-7743f1a8115a",
                    "name": "public",
                    "description": "fake appdynamics broker",
                    "metadata": {
                        "bullets": [],
                        "costs": [
                            {
                                "amount": {
                                    "usd": 0
                                },
                                "unit": "MONTHLY"
                            }
                        ],
                        "displayName": "public"
                    }
                }
            ]
        }
    ]
}


fake_credentials = {'account-access-key': 'test-key',
                    'account-name': 'test-account',
                    'host-name': 'test-sb-host',
                    'port': '1234',
                    'ssl-enabled': True
                    }


@broker.route("/")
def hello():
    return "Service Broker Up and Running"


@broker.route("/v2/catalog")
def catalog():
    return jsonify(fake_service)


@broker.route('/v2/service_instances/<instance_id>', methods=['PUT', 'DELETE', 'PATCH'])
def service_instances(instance_id):
    if request.method == 'PUT':
        return make_response(jsonify({}), 201)
    else:
        return jsonify({})


@broker.route('/v2/service_instances/<instance_id>/service_bindings/<binding_id>', methods=['PUT', 'DELETE'])
def bind_instances(instance_id, binding_id):
    if request.method == 'PUT':
        return make_response(jsonify({'credentials': fake_credentials}), 201)
    else:
        return jsonify({})


if __name__ == '__main__':
    broker.run(host='0.0.0.0', port=int(os.getenv('VCAP_APP_PORT', '5000')))
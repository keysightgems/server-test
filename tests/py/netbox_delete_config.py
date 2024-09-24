import pytest
import requests
from utils import common
import json
import logging

logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)

configData = common.config_data()

@pytest.mark.laas_sanity
def test_delete_config():
    """
    Test case going to configure the nodes and switches information on Netbox using the csv file.
    """
    _delete_apis(configData['list_apis'], configData)

def _delete_apis(url_path, config_data):
    """
    Internal method to delete the configured nodes and switches on Netbox.
    """
    auth_token = "Token {}".format(config_data['netbox_api_token'])
    headers = {'Authorization': auth_token, 'Content-Type': 'application/json'}
    for upath in url_path:
        base_url = config_data['netbox_api_url'] + upath
        response = requests.request("GET", base_url, headers=headers)
        data = response.json()["results"]
        new_url = base_url + "/"
        data=json.dumps(data)
        response = requests.request("DELETE", new_url, data=data, headers=headers)
        if response.status_code == 204:
            log.info(f"{url_path} config deleted successfully")
        else:
            log.info(f"Failed to delete {url_path} config: {response.text}")

import requests
import pytest
from utils import common

@pytest.mark.laas_sanity
def test_reset_node_interface_state():
    configData = common.config_data()
    auth_token = "Token {}".format(configData['netbox_api_token'])
    headers = {'Authorization': auth_token, 'Content-Type': 'application/json'}
    device_names = _get_nodes_names(configData['netbox_api_url'] + "dcim/devices/", headers)
    port_names = _get_interfaces_names(configData['netbox_api_url'] + "dcim/interfaces/", headers)
    assert _update_nodes(device_names, configData['netbox_api_url'], headers), f"Failed to update device state"
    assert _update_interfaces(port_names, configData['netbox_api_url'], headers), f"Failed to update interface state"

def _get_nodes_names(base_url, headers):
    device_names = []
    response = requests.request("GET", base_url, headers=headers)
    for resp in response.json()["results"]:
        if resp["role"]['name'].lower() == "dut":
            device_names.append(resp["name"])
    return device_names

def _get_interfaces_names(base_url, headers):
    port_names = []
    response = requests.request("GET", base_url, headers=headers)
    for resp in response.json()["results"]:
        if "state" in resp["custom_fields"] and (resp["custom_fields"]["state"]).lower() == "available":
            port_names.append(resp["name"])
        elif "state" in resp["custom_fields"] and (resp["custom_fields"]["state"]).lower() == "reserved":
            port_names.append(resp["name"])
        elif "state" in resp["custom_fields"] and (resp["custom_fields"]["state"]).lower() == "available":
            port_names.append(resp["name"])
        elif "state" in resp["custom_fields"] and (resp["custom_fields"]["state"]).lower() == "reserved":
            port_names.append(resp["name"])
        else:
            pass
    return port_names

def _update_nodes(device_names, base_url, headers):

    for deviceName in device_names:
        # import pdb;pdb.set_trace()
        url = f"{base_url}dcim/devices/?name={deviceName}"
        response = requests.request("GET", url, headers=headers)
        for resp in response.json()["results"]:
            update_data = {
                "name": resp["name"],
                "device_type": resp["device_type"]["id"],
                "custom_fields": {"state": "Available"},
            }
            response = requests.patch(resp["url"], json=update_data, headers=headers)
            if response.status_code != 200:
                assert False, f"Failed to update device state. Status code: {response.status_code}"
    return True

def _update_interfaces(port_names, base_url, headers):

    for portName in port_names:
        url = f"{base_url}dcim/interfaces/?name={portName}"
        response = requests.request("GET", url, headers=headers)
        for resp in response.json()["results"]:
            update_data = {"custom_fields": {"state": "Available"}}
            response = requests.patch(resp["url"], json=update_data, headers=headers)
            if response.status_code != 200:
                assert False, f"Failed to update device state. Status code: {response.status_code}"
    return True

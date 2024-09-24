import requests
import argparse
import concurrent.futures

def parse_args():
    parser = argparse.ArgumentParser(description='Reset node interface state')
    parser.add_argument('--netbox_api_token', required=True, help='NetBox API token')
    parser.add_argument('--netbox_api_url', required=True, help='NetBox API URL')
    return parser.parse_args()

def test_reset_node_interface_state(args):
    auth_token = "Token {}".format(args.netbox_api_token)
    headers = {'Authorization': auth_token, 'Content-Type': 'application/json'}
    device_url = "http://" + args.netbox_api_url + "/api/dcim/devices/"
    interface_url = "http://" + args.netbox_api_url + "/api/dcim/interfaces/"    
    device_names = _get_nodes_names(device_url, headers)
    port_names = _get_interfaces_names(interface_url, headers)
    
    with concurrent.futures.ThreadPoolExecutor() as executor:
        node_interface_state_results = list(executor.map(_update_nodes, device_names, [device_url]*len(device_names), [headers]*len(device_names)))
        interface_state_results = list(executor.map(_update_interfaces, port_names, [interface_url]*len(port_names), [headers]*len(port_names)))
    
    assert all(node_interface_state_results), f"Failed to update device state"
    assert all(interface_state_results), f"Failed to update interface state"

def _get_nodes_names(base_url, headers):
    device_names = []
    next_url = base_url
    while next_url:
        response = requests.request("GET", next_url, headers=headers)
        data = response.json()
        for resp in data["results"]:
            if resp["role"]['name'].lower() != "ate" and resp["role"]['name'].lower() != "l1s":
                device_names.append(resp["name"])
        next_url = data["next"]
    return device_names

def _get_interfaces_names(base_url, headers):
    port_names = []
    next_url = base_url
    while next_url:
        response = requests.request("GET", next_url, headers=headers)
        data = response.json()
        for resp in data["results"]:
            if "custom_fields" in resp and "state" in resp["custom_fields"]:
                if resp["custom_fields"]["state"].lower() == "reserved":
                    port_names.append(resp["name"])
        next_url = data["next"]
    return port_names

def _update_nodes(deviceName, base_url, headers):
    url = f"{base_url}?name={deviceName}"
    next_url = url
    while next_url:
        response = requests.request("GET", next_url, headers=headers)
        data = response.json()
        for resp in data["results"]:
            update_data = {
                "name": resp["name"],
                "device_type": resp["device_type"]["id"],
                "custom_fields": {"state": "Available"},
            }
            response = requests.patch(resp["url"], json=update_data, headers=headers)
            if response.status_code != 200:
                print(f"Failed to update device state. Status code: {response.status_code}")
                return False
        next_url = data["next"]
    return True

def _update_interfaces(portName, base_url, headers):
    url = f"{base_url}?name={portName}"
    next_url = url
    while next_url:
        response = requests.request("GET", next_url, headers=headers)
        data = response.json()
        for resp in data["results"]:
            update_data = {"custom_fields": {"state": "Available"}}
            response = requests.patch(resp["url"], json=update_data, headers=headers)
            if response.status_code != 200:
                print(f"Failed to update device state. Status code: {response.status_code}")
                return False
        next_url = data["next"]
    return True

if __name__ == "__main__":
    args = parse_args()
    test_reset_node_interface_state(args)
import opentestbed
import json, os, sys
import pytest
current_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(current_dir)
sys.path.append(parent_dir)
from utils import common

@pytest.mark.laas_sanity
def test_2nodes_1connection():
    """
    Test DUT to ATE direct with one connection and validate. 
    """
    configData = common.config_data()
    api = opentestbed.api(location=configData["location"], transport="http")

    testbed = opentestbed.Testbed()
    d1, d2 = testbed.devices.add(), testbed.devices.add()

    d1.id = "d1"
    d1.role = "DUT"
    d2.id = "d2"
    d2.role = "ATE"

    d1_port1 = d1.ports.add()
    d1_port1.id = "intf1"
    d1_port1.speed = d1_port1.S_400GB

    d2_port1 = d2.ports.add()
    d2_port1.id = "intf1"
    d2_port1.speed = d2_port1.S_400GB

    link1 = testbed.links.add()

    link1.src.device = d1.id
    link1.src.port = d1_port1.id
    link1.dst.device = d2.id
    link1.dst.port = d2_port1.id    
    output = api.reserve(testbed)
    outputDict = json.loads(output.testbed)
    testbedDict = testbed.serialize()
    validate_output(outputDict, testbedDict)

def validate_output(output, testbed):
    assert isinstance(output, dict), "Input data format incorrect"
    devices = output.get('devices', {})
    assert len(devices) == 2  #"Number of devices should be 2"
    device_ids = [device['id'] for device in devices.values()]
    assert len(set(device_ids)) == len(device_ids)  #"Device IDs should not match"
    # Validate links
    links = output.get('links', [])
    assert len(links) == 1
    for link in links:
        src_device = link['src']['device']
        dst_device = link['dst']['device']
        src_port = link['src']['port']
        dst_port = link['dst']['port']

        assert src_device != dst_device, "Source and destination devices should be different"
        assert src_port != dst_port, "Source and destination ports should be different"
        
        # Validate that ports exist in respective devices
        src_device_ports = []
        dst_device_ports = []
        for k, v in devices.items():
          if v['id'] == src_device:
              src_device_ports = [portvalue['attributes']['name'] for _, portvalue in v['ports'].items()]            
          elif v['id'] == dst_device:
              dst_device_ports = [portvalue['attributes']['name'] for _, portvalue in v['ports'].items()] 
          else:
              assert False, f"Failed to find Source and Destination Ports"
        assert src_port in src_device_ports, f"Source port {src_port} not found in source device {src_device}"
        assert dst_port in dst_device_ports, f"Destination port {dst_port} not found in destination device {dst_device}"
    # Validate Role and Speed from the input    
    testbedDict = json.loads(testbed)    
    inputRoleSpeed = {device['id']: [device['role'], device['ports'][0]['speed']] for device in testbedDict['devices']}
    outRoleSpeed = {}
    for device_id, device_info in output['devices'].items():
        role = device_info['attributes']['role']
        for port_id, port_info in device_info['ports'].items():
            speed = port_info['attributes']['speed']
            outRoleSpeed[device_id] = [role, speed]
    assert inputRoleSpeed == outRoleSpeed, f"Test case failed: Input and Response role/speed not matched. Input value: {inputRoleSpeed}, Output value: {outRoleSpeed}"
   

import opentestbed
import json, os, sys



    
api = opentestbed.api(location="http://10.39.70.169:8080", transport="http")

testbed = opentestbed.Testbed()
d1, d2, d3 = testbed.devices.add(), testbed.devices.add(), testbed.devices.add()

d1.id = "d1"
d1.role = "DUT"
d2.id = "d2"
d2.role = "DUT"
d3.id = "d3"
d3.role = "ATE"

d1_port1 = d1.ports.add()
d1_port1.id = "intf1"
d1_port1.speed = d1_port1.S_400GB

d2_port1 = d2.ports.add()
d2_port1.id = "intf1"
d2_port1.speed = d2_port1.S_400GB

d3_port1 = d3.ports.add()
d3_port1.id = "intf1"
d3_port1.speed = d3_port1.S_400GB    
d3_port2 = d3.ports.add()
d3_port2.id = "intf2"
d3_port2.speed = d3_port2.S_400GB

link1 = testbed.links.add()
link1.src.device = d1.id
link1.src.port = d1_port1.id
link1.dst.device = d3.id
link1.dst.port = d3_port1.id    
link2 = testbed.links.add()
link2.src.device = d2.id
link2.src.port = d2_port1.id
link2.dst.device = d3.id
link2.dst.port = d3_port2.id

output = api.reserve(testbed)
print(output)
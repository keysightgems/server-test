import json, os
import paramiko

# def config_data():
#     with open('config.json') as f:
#         return json.load(f)
    
def config_data():
    config_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'config.json')
    with open(config_path) as f:
        return json.load(f)
 
def ssh_client():
    configData = config_data()
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh_client.connect(hostname=configData['ssh_domain'],
                       port=configData['ssh_port'],
                       username=configData['ssh_user'],
                       password=configData['ssh_pass'])
    return ssh_client
    # ssh_client.close()
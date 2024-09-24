import pytest
from utils import common

configData = common.config_data()
sshClient = common.ssh_client()

@pytest.mark.laas_sanity
def test_netbox_deployment():
    """
    Test case will deploy the Netbox and setup make it ready.
    """    
    sftp_client = sshClient.open_sftp()
    for local_file in configData['files']:
        local_path = f'{local_file}'
        remote_path = f'/root/{local_file}'
        sftp_client.put(local_path, remote_path)
    sftp_client.close()
    stdin, stdout, stderr = sshClient.exec_command('chmod +x ./deploy_netbox.sh')
    stdout.channel.recv_exit_status()
    stdin, stdout, stderr = sshClient.exec_command('./deploy_netbox.sh')
    stdout.read().decode('utf-8')
    stderr.read().decode('utf-8')


    

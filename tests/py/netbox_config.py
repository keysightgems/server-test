import pytest
import requests
from utils import common
import os
from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from webdriver_manager.chrome import ChromeDriverManager
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.common.exceptions import TimeoutException
import json
import logging
import time

logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)

configData = common.config_data()

@pytest.mark.laas_sanity
def test_add_config():
    """
    Test case going to configure the nodes and switches information on Netbox using the csv file.
    """
    _load_csv(configData['dict_apis'], configData)
    # _delete_apis(configData['list_apis'], configData)

def _load_csv(apis_dict, config_data):
    """
    Internal method, using to import all the configuration csv files.
    """
    current_dir = os.getcwd()
    driver = webdriver.Chrome(service=Service(ChromeDriverManager().install()))
    # import pdb;pdb.set_trace()
    url = config_data['netbox_selenium_url'] + "login/?next=/"
    driver.get(url)
    driver.maximize_window()
    WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, config_data['login_user'])))
    username_field = driver.find_element(By.XPATH, config_data['login_user'])
    username_field.send_keys(config_data['netbox_username'])
    WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, config_data['login_pass'])))
    password_field = driver.find_element(By.XPATH, config_data['login_pass'])
    password_field.send_keys(config_data['netbox_pass'])
    driver.find_element(By.XPATH, config_data['login_submit']).click()
    assert "Logged in" in driver.page_source
    for key, value in apis_dict.items():
        base_url = config_data['netbox_selenium_url'] + key
        driver.get(base_url)
        WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, value[1])))
        driver.find_element(By.XPATH, value[1]).click()
        WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, config_data['upload_tab'])))
        driver.find_element(By.XPATH, config_data['upload_tab']).click()
        WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.ID, config_data['upoload_input_id'])))
        file_input = driver.find_element(By.ID, config_data['upoload_input_id'])
        file_path = os.path.join(current_dir, value[0])
        file_input.send_keys(file_path)
        WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, config_data['upload_submit'])))
        driver.find_element(By.XPATH, config_data['upload_submit']).click()  
        time.sleep(3)
        message_text = driver.find_element(By.XPATH, config_data['import_success']).text
        if "Imported" in message_text:
            assert True, f"Successfully imported {key}, Success Message: {message_text}"
        else:
            assert False, f"Failed to upload {key}, please check the csv input and try again"                 
    driver.quit()

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

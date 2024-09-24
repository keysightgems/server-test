package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

type DeviceResponse struct {
	Results []Device `json:"results"`
	Next    string   `json:"next"`
}

type Device struct {
	Name       string `json:"name"`
	DeviceType struct {
		ID int `json:"id"`
	} `json:"device_type"`
	Role struct {
		Name string `json:"name"`
	} `json:"role"`
	URL          string `json:"url"`
	CustomFields struct {
		State string `json:"state"`
	} `json:"custom_fields"`
}

type InterfaceResponse struct {
	Results []Interface `json:"results"`
	Next    string      `json:"next"`
}

type Interface struct {
	Name         string `json:"name"`
	CustomFields struct {
		State string `json:"state"`
	} `json:"custom_fields"`
	URL string `json:"url"`
}

func main() {
	netboxAPIToken := flag.String("netbox_api_token", "", "NetBox API token")
	netboxAPIURL := flag.String("netbox_api_url", "", "NetBox API URL")
	flag.Parse()

	if *netboxAPIToken == "" || *netboxAPIURL == "" {
		log.Fatal("NetBox API token and URL must be provided")
	}

	headers := map[string]string{
		"Authorization": "Token " + *netboxAPIToken,
		"Content-Type":  "application/json",
	}
	deviceURL := "http://" + *netboxAPIURL + "/api/dcim/devices/"
	interfaceURL := "http://" + *netboxAPIURL + "/api/dcim/interfaces/"

	deviceNames, err := getNodesNames(deviceURL, headers)
	if err != nil {
		log.Fatalf("Failed to get device names: %v", err)
	}
	portNames, err := getInterfacesNames(interfaceURL, headers)
	if err != nil {
		log.Fatalf("Failed to get interface names: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	nodeInterfaceStateResults := make([]bool, len(deviceNames))
	interfaceStateResults := make([]bool, len(portNames))

	wg.Add(len(deviceNames))
	for i, deviceName := range deviceNames {
		go func(i int, deviceName string) {
			defer wg.Done()
			result := updateNodes(deviceName, deviceURL, headers)
			mu.Lock()
			nodeInterfaceStateResults[i] = result
			mu.Unlock()
		}(i, deviceName)
	}

	wg.Add(len(portNames))
	for i, portName := range portNames {
		go func(i int, portName string) {
			defer wg.Done()
			result := updateInterfaces(portName, interfaceURL, headers)
			mu.Lock()
			interfaceStateResults[i] = result
			mu.Unlock()
		}(i, portName)
	}

	wg.Wait()

	for _, result := range nodeInterfaceStateResults {
		if !result {
			log.Fatal("Failed to update device state")
		}
	}
	for _, result := range interfaceStateResults {
		if !result {
			log.Fatal("Failed to update interface state")
		}
	}
}

func getNodesNames(baseURL string, headers map[string]string) ([]string, error) {
	var deviceNames []string
	nextURL := baseURL
	for nextURL != "" {
		resp, err := makeRequest("GET", nextURL, headers, nil)
		if err != nil {
			return nil, err
		}
		var data DeviceResponse
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, err
		}
		for _, device := range data.Results {
			if device.Role.Name != "ate" && device.Role.Name != "l1s" {
				deviceNames = append(deviceNames, device.Name)
			}
		}
		nextURL = data.Next
	}
	return deviceNames, nil
}

func getInterfacesNames(baseURL string, headers map[string]string) ([]string, error) {
	var portNames []string
	nextURL := baseURL
	for nextURL != "" {
		resp, err := makeRequest("GET", nextURL, headers, nil)
		if err != nil {
			return nil, err
		}
		var data InterfaceResponse
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, err
		}
		for _, iface := range data.Results {
			if iface.CustomFields.State == "reserved" {
				portNames = append(portNames, iface.Name)
			}
		}
		nextURL = data.Next
	}
	return portNames, nil
}

func updateNodes(deviceName, baseURL string, headers map[string]string) bool {
	url := fmt.Sprintf("%s?name=%s", baseURL, deviceName)
	nextURL := url
	for nextURL != "" {
		resp, err := makeRequest("GET", nextURL, headers, nil)
		if err != nil {
			log.Printf("Failed to get device details: %v", err)
			return false
		}
		var data DeviceResponse
		if err := json.Unmarshal(resp, &data); err != nil {
			log.Printf("Failed to unmarshal device response: %v", err)
			return false
		}
		for _, device := range data.Results {
			updateData := map[string]interface{}{
				"name":        device.Name,
				"device_type": device.DeviceType.ID,
				"custom_fields": map[string]string{
					"state": "Available",
				},
			}
			updateDataJSON, err := json.Marshal(updateData)
			if err != nil {
				log.Printf("Failed to marshal update data: %v", err)
				return false
			}
			_, err = makeRequest("PATCH", device.URL, headers, updateDataJSON)
			if err != nil {
				log.Printf("Failed to update device state: %v", err)
				return false
			}
		}
		nextURL = data.Next
	}
	return true
}

func updateInterfaces(portName, baseURL string, headers map[string]string) bool {
	url := fmt.Sprintf("%s?name=%s", baseURL, portName)
	nextURL := url
	for nextURL != "" {
		resp, err := makeRequest("GET", nextURL, headers, nil)
		if err != nil {
			log.Printf("Failed to get interface details: %v", err)
			return false
		}
		var data InterfaceResponse
		if err := json.Unmarshal(resp, &data); err != nil {
			log.Printf("Failed to unmarshal interface response: %v", err)
			return false
		}
		for _, iface := range data.Results {
			updateData := map[string]interface{}{
				"custom_fields": map[string]string{
					"state": "Available",
				},
			}
			updateDataJSON, err := json.Marshal(updateData)
			if err != nil {
				log.Printf("Failed to marshal update data: %v", err)
				return false
			}
			_, err = makeRequest("PATCH", iface.URL, headers, updateDataJSON)
			if err != nil {
				log.Printf("Failed to update interface state: %v", err)
				return false
			}
		}
		nextURL = data.Next
	}
	return true
}

func makeRequest(method, url string, headers map[string]string, body []byte) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

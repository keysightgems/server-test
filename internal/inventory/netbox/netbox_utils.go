package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"keysight/laas/controller/config"
	"keysight/laas/controller/internal/profile"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// NETBOX_URL = "http://10.39.70.169:8000/api/"
	// TOKEN      = "3a0d77d5d033af6a3e2c168f2c9e2cb6d81082b1"
	HEADERS = "application/json"
)

var httpClient = &http.Client{}

func createRequest(method, url string, TOKEN string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+TOKEN)
	req.Header.Set("Content-Type", HEADERS)
	return req, nil
}

func performRequest(req *http.Request) (*http.Response, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func updateDevicesData(jsonFile string, NETBOX_URL string, TOKEN string, userID string, releaseState map[string][]map[string]interface{}) error {
	defer profile.LogFuncDuration(time.Now(), "updateDevicesData", "", "inventory")
	// Read data from JSON file
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %v, Error: %v", jsonFile, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON file: %v", err)
	}
	deviceNames := map[string]string{}
	for _, value := range data {
		if v, ok := value.(map[string]interface{}); ok {
			for _, v := range v {
				if vv, ok := v.(map[string]interface{}); ok {
					name := vv["id"].(string)
					if attributes, ok := vv["attributes"].(map[string]interface{}); ok {
						model := attributes["role"].(string)
						deviceNames[name] = model
					}

				}
			}
		}
	}
	portNames := []map[string]string{} // Define portNames as a slice of maps
	devices, devicesOK := data["devices"].(map[string]interface{})
	if devicesOK {
		for _, device := range devices {
			deviceMap, deviceMapOK := device.(map[string]interface{})
			if !deviceMapOK {
				continue
			}
			deviceID, idOK := deviceMap["id"].(string) // Get device ID
			if !idOK {
				continue
			}
			ports, portsOK := deviceMap["ports"].(map[string]interface{})
			if !portsOK {
				continue
			}
			for _, port := range ports {
				portMap, portMapOK := port.(map[string]interface{})
				if !portMapOK {
					continue
				}
				portAttributes, attributesOK := portMap["attributes"].(map[string]interface{})
				if !attributesOK {
					continue
				}
				// Access port attributes
				name, nameOK := portAttributes["name"].(string)
				if nameOK {
					// Create a map to store device ID and port name
					devicePort := map[string]string{
						deviceID: name,
					}
					// Append the devicePort map to portNames slice
					portNames = append(portNames, devicePort)
				}
			}
		}
	}
	client := httpClient
	for deviceName, model := range deviceNames {
		url := fmt.Sprintf("%sdcim/devices/?name=%s", NETBOX_URL, deviceName)
		req, err := createRequest("GET", url, TOKEN, nil)
		if err != nil {
			return fmt.Errorf("get request failed: %v, Error: %v", url, err)
		}
		response, err := performRequest(req)
		if err != nil {
			return fmt.Errorf("failed to get response: %v, Error: %v", url, err)
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response url: %v, Error: %v", url, err)
		}

		var deviceDict map[string]interface{}
		if err := json.Unmarshal(body, &deviceDict); err != nil {
			return fmt.Errorf("failed to unmarshal deviceDict: %v", err)
		}
		results := deviceDict["results"].([]interface{})
		if len(results) > 0 {
			deviceDict := results[0].(map[string]interface{})
			if strings.ToLower(model) != "ate" && strings.ToLower(model) != "l1s" && strings.ToLower(deviceDict["custom_fields"].(map[string]interface{})["state"].(string)) != "reserved" {
				if strings.EqualFold(deviceDict["name"].(string), deviceName) {
					deviceURL := deviceDict["url"].(string)
					updateData := map[string]interface{}{
						"name":        deviceDict["name"],
						"device_type": deviceDict["device_type"].(map[string]interface{})["id"],
						"custom_fields": map[string]interface{}{
							"session_id": userID,
							"state":      "Reserved",
						},
					}
					// Store the updateData and deviceURL in the global variable
					deviceUpdate := map[string]interface{}{
						deviceURL: updateData,
					}
					// Ensure userID is a string
					if _, exists := releaseState[userID]; !exists {
						releaseState[userID] = []map[string]interface{}{deviceUpdate}
					} else {
						releaseState[userID] = append(releaseState[userID], deviceUpdate)
					}
					updateDataJSON, err := json.Marshal(updateData)
					if err != nil {
						return fmt.Errorf("failed Json marshal with updateData: %v", err)
					}
					req, err := http.NewRequest("PATCH", deviceURL, bytes.NewBuffer(updateDataJSON))
					if err != nil {
						return fmt.Errorf("failed to patch the data with url: %v, Error: %v", deviceURL, err)
					}
					req.Header.Set("Authorization", "Token "+TOKEN)
					req.Header.Set("Content-Type", HEADERS)
					response, err := client.Do(req)
					if err != nil {
						return fmt.Errorf("failed to get response for deviceURL: %v, Error: %v", deviceURL, err)
					}
					defer response.Body.Close()
					if response.StatusCode != http.StatusOK {
						return fmt.Errorf("error updating device details, status code: %v", response.StatusCode)
					}
				} else {
					return fmt.Errorf("failed to find the device: %v", deviceName)
				}

			} else if strings.ToLower(model) != "ate" && strings.ToLower(model) != "l1s" && strings.ToLower(deviceDict["custom_fields"].(map[string]interface{})["state"].(string)) == "reserved" && strings.ToLower(deviceDict["custom_fields"].(map[string]interface{})["session_id"].(string)) == userID {
				if strings.EqualFold(deviceDict["name"].(string), deviceName) {
					deviceURL := deviceDict["url"].(string)
					updateData := map[string]interface{}{
						"name":        deviceDict["name"],
						"device_type": deviceDict["device_type"].(map[string]interface{})["id"],
						"custom_fields": map[string]interface{}{
							"state":      "Available",
							"session_id": "",
						},
					}
					updateDataJSON, err := json.Marshal(updateData)
					if err != nil {
						return fmt.Errorf("failed Json marshal with updateData: %v", err)
					}
					req, err := http.NewRequest("PATCH", deviceURL, bytes.NewBuffer(updateDataJSON))
					if err != nil {
						return fmt.Errorf("failed to patch the data with url: %v, Error: %v", deviceURL, err)
					}
					req.Header.Set("Authorization", "Token "+TOKEN)
					req.Header.Set("Content-Type", HEADERS)
					response, err := client.Do(req)
					if err != nil {
						return fmt.Errorf("failed to get response for deviceURL: %v, Error: %v", deviceURL, err)
					}
					defer response.Body.Close()
					if response.StatusCode != http.StatusOK {
						return fmt.Errorf("error updating device details, status code: %v", response.StatusCode)
					}
				} else {
					return fmt.Errorf("failed to find the device: %v", deviceName)
				}
			} else if strings.ToLower(model) != "ate" && strings.ToLower(model) != "l1s" && strings.ToLower(deviceDict["custom_fields"].(map[string]interface{})["state"].(string)) == "reserved" && strings.ToLower(deviceDict["custom_fields"].(map[string]interface{})["session_id"].(string)) != userID {
				return fmt.Errorf("failed to update device details")
			}
		}
	}
	for _, portName := range portNames {
		for deviceID, name := range portName {
			url := fmt.Sprintf("%sdcim/interfaces/?name=%s", NETBOX_URL, name)
			req, err := createRequest("GET", url, TOKEN, nil)
			if err != nil {
				return fmt.Errorf("get request failed: %v, Error: %v", url, err)
			}
			response, err := performRequest(req)
			if err != nil {
				return fmt.Errorf("failed to get response: %v, Error: %v", url, err)
			}
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("failed to read response url: %v, Error: %v", url, err)
			}

			var portDict map[string]interface{}
			if err := json.Unmarshal(body, &portDict); err != nil {
				return fmt.Errorf("failed to unmarshal deviceDict: %v", err)
			}
			results := portDict["results"].([]interface{})
			for _, result := range results {
				portDict := result.(map[string]interface{})
				var devicename string
				if deviceData, ok := portDict["device"].(map[string]interface{}); ok {
					devicename = deviceData["name"].(string)
				}
				if strings.EqualFold(portDict["name"].(string), name) && strings.EqualFold(devicename, deviceID) && strings.ToLower(portDict["custom_fields"].(map[string]interface{})["state"].(string)) != "reserved" {
					portURL := portDict["url"].(string)
					updateData := map[string]interface{}{
						"custom_fields": map[string]interface{}{
							"session_id": userID,
							"state":      "Reserved",
						},
					}
					// Store the updateData and deviceURL in the global variable
					deviceUpdate := map[string]interface{}{
						portURL: updateData,
					}
					// Ensure userID is a string
					if _, exists := releaseState[userID]; !exists {
						releaseState[userID] = []map[string]interface{}{deviceUpdate}
					} else {
						releaseState[userID] = append(releaseState[userID], deviceUpdate)
					}
					updateDataJSON, err := json.Marshal(updateData)
					if err != nil {
						return fmt.Errorf("failed Json marshal with updateData: %v", err)
					}
					req, err := http.NewRequest("PATCH", portURL, bytes.NewBuffer(updateDataJSON))
					if err != nil {
						return fmt.Errorf("failed to patch the data with url: %v Error: %v", portURL, err)
					}
					req.Header.Set("Authorization", "Token "+TOKEN)
					req.Header.Set("Content-Type", HEADERS)
					response, err := client.Do(req)
					if err != nil {
						return fmt.Errorf("failed to get response for portURL: %v Error: %v", portURL, err)
					}
					defer response.Body.Close()
					if response.StatusCode != http.StatusOK {
						return fmt.Errorf("error updating port details, status code: %v", response.StatusCode)
					}
				} else if strings.EqualFold(portDict["name"].(string), name) && strings.EqualFold(devicename, deviceID) && strings.ToLower(portDict["custom_fields"].(map[string]interface{})["state"].(string)) == "reserved" && strings.ToLower(portDict["custom_fields"].(map[string]interface{})["session_id"].(string)) == userID {
					portURL := portDict["url"].(string)
					updateData := map[string]interface{}{
						"custom_fields": map[string]interface{}{
							"state":      "Available",
							"session_id": "",
						},
					}
					updateDataJSON, err := json.Marshal(updateData)
					if err != nil {
						return fmt.Errorf("failed Json marshal with updateData: %v", err)
					}
					req, err := http.NewRequest("PATCH", portURL, bytes.NewBuffer(updateDataJSON))
					if err != nil {
						return fmt.Errorf("failed to patch the data with url: %v Error: %v", portURL, err)
					}
					req.Header.Set("Authorization", "Token "+TOKEN)
					req.Header.Set("Content-Type", HEADERS)
					response, err := client.Do(req)
					if err != nil {
						return fmt.Errorf("failed to get response for portURL: %v Error: %v", portURL, err)
					}
					defer response.Body.Close()
					if response.StatusCode != http.StatusOK {
						return fmt.Errorf("error updating port details, status code: %v", response.StatusCode)
					}
				} else if strings.EqualFold(portDict["name"].(string), name) && strings.EqualFold(devicename, deviceID) && strings.ToLower(portDict["custom_fields"].(map[string]interface{})["state"].(string)) == "reserved" && strings.ToLower(portDict["custom_fields"].(map[string]interface{})["session_id"].(string)) != userID {
					return fmt.Errorf("failed to update interface details")
				}
			}
		}
	}
	return nil
	// Check if the file exists before attempting to delete
	// if _, err := os.Stat(jsonFile); err == nil {
	// 	// Delete the file
	// 	err := os.Remove(jsonFile)
	// 	if err != nil {
	// 		log.Fatalf("Error deleting file: %v\n", err)
	// 	}
	// 	log.Printf("File '%s' deleted successfully.\n", jsonFile)
	// } else {
	// 	log.Printf("File '%s' does not exist.\n", jsonFile)
	// }
}

func getDeviceDetails(deviceName string, NETBOX_URL string, TOKEN string) map[string]interface{} {
	url := fmt.Sprintf("%sdcim/devices/?name=%s", NETBOX_URL, deviceName)
	req, err := createRequest("GET", url, TOKEN, nil)
	if err != nil {
		log.Fatal().Msgf("Get request failed: %v Error: %v", url, err)
	}
	response, err := performRequest(req)
	if err != nil {
		log.Fatal().Msgf("Failed to get response with the url: %v Error: %v", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var result map[string]interface{}
		_ = json.NewDecoder(response.Body).Decode(&result)
		deviceDetails := result["results"].([]interface{})[0].(map[string]interface{})
		return deviceDetails
	} else {
		log.Error().Interface("Status Code", response.StatusCode).Msg("Error fetching device details")
		return nil
	}
}

func getDevicesDetails(NETBOX_URL string, TOKEN string) []string {
	defer profile.LogFuncDuration(time.Now(), "getDevicesDetails", "", "inventory")

	var allDevices []string
	offset := 0
	limit := 1000
	for {
		url := fmt.Sprintf("%sdcim/devices/?offset=%d&limit=%d", NETBOX_URL, offset, limit)
		req, err := createRequest("GET", url, TOKEN, nil)
		if err != nil {
			log.Fatal().Msgf("Get request failed: %v, Error: %v", url, err)
		}
		response, err := performRequest(req)
		if err != nil {
			log.Fatal().Msgf("Failed to get response with the url: %v, Error: %v", url, err)
		}
		defer response.Body.Close()
		if response.StatusCode == http.StatusOK {
			var result map[string]interface{}
			_ = json.NewDecoder(response.Body).Decode(&result)
			deviceDetails := result["results"].([]interface{})
			for _, device := range deviceDetails {
				allDevices = append(allDevices, device.(map[string]interface{})["name"].(string))
			}

			// Check if there are more pages
			nextPage := result["next"]
			if nextPage == nil {
				break
			}

			// Update the offset for the next page
			offset += limit
		} else {
			log.Error().Interface("Status Code", response.StatusCode).Msg("Error fetching device details")
			break
		}
	}

	log.Debug().Interface("All Devices", allDevices).Msg("Obtained devices details")
	return allDevices
}

func getInterfacesDetails(deviceName string, NETBOX_URL string, TOKEN string) []map[string]interface{} {
	defer profile.LogFuncDuration(time.Now(), "getInterfacesDetails", "", "inventory")

	url := fmt.Sprintf("%sdcim/interfaces/?device=%s", NETBOX_URL, deviceName)

	req, err := createRequest("GET", url, TOKEN, nil)
	if err != nil {
		log.Fatal().Msgf("Get request failed: %v, Error: %v", url, err)
	}
	response, err := performRequest(req)
	if err != nil {
		log.Fatal().Msgf("Failed to get response with the url: %v, Error: %v", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var result map[string]interface{}
		_ = json.NewDecoder(response.Body).Decode(&result)
		interfaceDetails := result["results"].([]interface{})
		var interfaceList []map[string]interface{}
		for _, iface := range interfaceDetails {
			interfaceList = append(interfaceList, iface.(map[string]interface{}))
		}
		log.Debug().Interface("Interfaces", interfaceList).Msg("Obtained Interfaces details")
		return interfaceList
	} else {
		log.Error().Interface("Status Code", response.StatusCode).Msg("Error fetching interface details")
		return nil
	}
}

func getEachInterfaceDetails(NETBOX_URL string, TOKEN string) []map[string]interface{} {
	var allInterfaces []map[string]interface{}
	offset := 0
	limit := 1000
	for {
		url := fmt.Sprintf("%sdcim/interfaces/?offset=%d&limit=%d", NETBOX_URL, offset, limit)
		req, err := createRequest("GET", url, TOKEN, nil)
		if err != nil {
			log.Fatal().Msgf("Get request failed: %v, Error: %v", url, err)
		}

		response, err := performRequest(req)
		if err != nil {
			log.Fatal().Msgf("Failed to get response with the url: %v, Error: %v", url, err)
		}
		defer response.Body.Close()
		if response.StatusCode == http.StatusOK {
			var result map[string]interface{}
			_ = json.NewDecoder(response.Body).Decode(&result)
			interfaceDetails := result["results"].([]interface{})
			for _, iface := range interfaceDetails {
				allInterfaces = append(allInterfaces, iface.(map[string]interface{}))
			}

			// Check if there are more pages
			nextPage := result["next"]
			if nextPage == nil {
				break
			}

			// Update the offset for the next page
			offset += limit
		} else {
			log.Error().Interface("Status Code", response.StatusCode).Msg("Error fetching interface details")
			break
		}
	}
	return allInterfaces
}

func getValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]interface{}:
		value, exists := v["value"].(string)
		if exists {
			return value
		}
		return "val"
	default:
		return "val"
	}
}
func removeKeys(input map[string]interface{}, keysToRemove []string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range input {
		// Check if the key is not in the keysToRemove list
		if !contains(keysToRemove, key) {
			result[key] = value
		}
	}
	return result
}

// contains checks if a given string exists in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getDevicesData(netboxApiURL string, netboxApiToken string) []byte {
	defer profile.LogFuncDuration(time.Now(), "getDevicesData", "", "inventory")

	deviceList := getDevicesDetails(netboxApiURL, netboxApiToken)
	var listOfDeviceDicts []map[string]interface{}
	for _, deviceName := range deviceList {
		interfaceDict := make([]map[string]interface{}, 0)
		deviceDetails := getDeviceDetails(deviceName, netboxApiURL, netboxApiToken)

		if deviceDetails["interface_count"].(float64) > 0 {
			interfaceDetails := getInterfacesDetails(deviceName, netboxApiURL, netboxApiToken)
			for _, iface := range interfaceDetails {
				if iface["device"].(map[string]interface{})["name"].(string) == deviceName {
					if iface["speed"] != nil {
						speed := iface["speed"].(float64)
						switch speed {
						case 1000000:
							iface["speed"] = "S_1GB"
						case 5000000:
							iface["speed"] = "S_5GB"
						case 10000000:
							iface["speed"] = "S_10GB"
						case 25000000:
							iface["speed"] = "S_25GB"
						case 40000000:
							iface["speed"] = "S_40GB"
						case 50000000:
							iface["speed"] = "S_50GB"
						case 100000000:
							iface["speed"] = "S_100GB"
						case 200000000:
							iface["speed"] = "S_200GB"
						case 400000000:
							iface["speed"] = "S_400GB"
						}
					}
					pmdValue := getValidValue(iface, "custom_fields.pmd")
					if pmdValue == "" || pmdValue == "null" {
						pmdValue = "PMD_UNSPECIFIED"
					}
					customFields, ok := iface["custom_fields"].(map[string]interface{})
					if !ok {
						log.Fatal().Msgf("custom_fields is not a map[string]interface{}")
					}
					// Convert custom field keys and values to lowercase except "pmd" and "speed"
					for key, value := range customFields {
						if key == "pmd" || key == "speed" {
							continue
						}
						lowerKey := strings.ToLower(key)
						switch v := value.(type) {
						case string:
							customFields[lowerKey] = strings.ToLower(v)
						default:
							customFields[lowerKey] = value
						}
						// Remove the original key if it differs in case
						if lowerKey != key {
							delete(customFields, key)
						}
					}
					speedValue := iface["speed"]
					if iface["speed"] == "" || iface["speed"] == "null" || iface["speed"] == nil {
						speedValue = "SPEED_UNSPECIFIED"
					}
					// Update the "pmd" field if necessary
					if pmd, exists := customFields["pmd"]; !exists || pmd == nil || pmd == "null" {
						customFields["pmd"] = "PMD_UNSPECIFIED"
					}
					interfaceDict = append(interfaceDict, map[string]interface{}{
						"id":          iface["name"],
						"name":        iface["name"],
						"speed":       speedValue,
						"pmd":         pmdValue,
						"transceiver": strings.ToLower(getValidValue(iface, "custom_fields.transceiver")),
						"attributes":  customFields,
					})
				}
			}
		}
		if deviceDetails != nil {
			if len(interfaceDict) == 0 {
				interfaceDict = append(interfaceDict, map[string]interface{}{})
			}
			inventoryDeviceAttr1 := deviceDetails["custom_fields"].(map[string]interface{})
			keysToRemove := []string{"Connection", "Credential", "Handle_Name", "Via", "Image"}
			inventoryDeviceAttr := removeKeys(inventoryDeviceAttr1, keysToRemove)
			addr := getValidValue(deviceDetails, "primary_ip.address")
			rol := getValidValue(deviceDetails, "role.name")
			inventoryDeviceAttr["address"] = addr
			inventoryDeviceAttr["devicetype"] = rol
			for key, value := range inventoryDeviceAttr {
				inventoryDeviceAttr[key] = replaceNilWithNull(value)
			}
			// Convert custom field keys and values to lowercase except "pmd" and "speed"
			for key, value := range inventoryDeviceAttr {
				if key == "role" {
					continue
				}
				lowerKey := strings.ToLower(key)
				switch v := value.(type) {
				case string:
					inventoryDeviceAttr[lowerKey] = strings.ToLower(v)
				default:
					inventoryDeviceAttr[lowerKey] = value
				}
				// Remove the original key if it differs in case
				if lowerKey != key {
					delete(inventoryDeviceAttr, key)
				}
			}
			deviceData := map[string]interface{}{
				"Id":   deviceDetails["id"],
				"Name": deviceDetails["name"],
				"Role": rol,
				// "DeviceType":    getValidValue(deviceDetails, "device_type.model"),
				"DeviceType": getValidValue(deviceDetails, "custom_fields.Model"),
				// "Manufacturer":  getValidValue(deviceDetails, "device_type.manufacturer.name"),
				"Vendor":        getValidValue(deviceDetails, "custom_fields.Vendor"),
				"Platform":      getValidValue(deviceDetails, "platform.name"),
				"Image":         getValidValue(deviceDetails, "custom_fields.Image"),
				"State":         getValidValue(deviceDetails, "custom_fields.State"),
				"Connec":        getValidValue(deviceDetails, "custom_fields.Connection"),
				"Cred":          getValidValue(deviceDetails, "custom_fields.Credential"),
				"HandleName":    getValidValue(deviceDetails, "custom_fields.Handle_Name"),
				"Via":           getValidValue(deviceDetails, "custom_fields.Via"),
				"inventoryAttr": inventoryDeviceAttr,
				"interfaces":    interfaceDict,
			}
			listOfDeviceDicts = append(listOfDeviceDicts, deviceData)
		} else {
			log.Error().Interface("Device not found", deviceName).Msg("")
		}
	}
	result, err := json.MarshalIndent(listOfDeviceDicts, "", "  ")
	if err != nil {
		log.Fatal().Msgf("Failed to Marshal listOfDeviceDicts: %v", err)
	}
	return result
}

func replaceNilWithNull(value interface{}) interface{} {
	if value == nil {
		return "null"
	}
	return value
}

func getValidValue(data map[string]interface{}, path string) string {
	state := getFieldValue(data, path)
	return getValue(state)
}

func getFieldValue(data map[string]interface{}, path string) interface{} {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		// Case-insensitive key comparison
		found := false
		for actualKey, value := range current {
			if strings.EqualFold(actualKey, key) {

				// If the key is found, update 'current' and set found to true
				if v, ok := value.(map[string]interface{}); ok {
					current, found = v, true
				} else if v, ok := value.(string); ok {
					current, found = map[string]interface{}{"value": v}, true
				} else {
					current, found = nil, true
				}
				break
			}
		}

		// If the key is not found or value is nil, return "null"
		if !found || current == nil {
			return "null"
		}
	}

	return current
}

func getDevicesLinks(netboxApiURL string, netboxApiToken string) []byte {
	defer profile.LogFuncDuration(time.Now(), "getDevicesLinks", "", "inventory")

	interfaceDetails := getEachInterfaceDetails(netboxApiURL, netboxApiToken)
	links := make([]map[string]string, 0)
	for _, iface := range interfaceDetails {
		srcDeviceName := iface["device"].(map[string]interface{})["name"].(string)
		src := srcDeviceName + ":" + iface["name"].(string)
		var dst string
		if linkPeers, ok := iface["link_peers"].([]interface{}); ok && len(linkPeers) > 0 {
			for _, peer := range linkPeers {
				peerMap := peer.(map[string]interface{})
				dstDeviceName := peerMap["device"].(map[string]interface{})["name"].(string)
				dst = dstDeviceName + ":" + peerMap["name"].(string)
				if src != "" && dst != "" {
					links = append(links, map[string]string{"src": src, "dst": dst})
				}
			}
		}
	}
	// Deduplicate links
	uniqueLinks := make([]map[string]string, 0)
	seenLinks := make(map[string]struct{})
	for _, link := range links {
		key := link["src"] + link["dst"]
		if _, seen := seenLinks[key]; !seen {
			uniqueLinks = append(uniqueLinks, link)
			seenLinks[key] = struct{}{}
		}
	}
	log.Debug().Interface("Devices links", uniqueLinks).Msg("Obtained Devices links")
	jsonStr, err := json.MarshalIndent(uniqueLinks, "", "  ")
	if err != nil {
		log.Fatal().Msgf("Failed to Marshal uniqueLinks: %v", err)
	}
	return jsonStr
}

func updateNodeState(releaseState map[string][]map[string]interface{}, user_id string) error {
	defer profile.LogFuncDuration(time.Now(), "updateNodeState", "", "inventory")
	for userId, nodes := range releaseState {
		if userId == user_id {
			for _, details := range nodes {
				for url, data := range details {
					updatedData, err := stateUpdate(data)
					if err != nil {
						return fmt.Errorf("failed to get updated Json Data: %v", err)
					}
					updateDataJSON, err := json.Marshal(updatedData)
					if err != nil {
						return fmt.Errorf("failed Json marshal with updateData: %v", err)
					}
					req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(updateDataJSON))
					if err != nil {
						return fmt.Errorf("failed to patch the data with url: %v Error: %v", url, err)
					}
					req.Header.Set("Authorization", "Token "+*config.Config.NetboxUserToken)
					req.Header.Set("Content-Type", HEADERS)
					response, err := httpClient.Do(req)
					if err != nil {
						return fmt.Errorf("failed to get response for portURL: %v Error: %v", url, err)
					}
					defer response.Body.Close()
					if response.StatusCode != http.StatusOK {
						return fmt.Errorf("error updating port details, status code: %v", response.StatusCode)
					}
				}
			}
		}
	}
	return nil
}

// Function that accepts an interface{} and updates its state
func stateUpdate(data interface{}) (map[string]interface{}, error) {
	// Assert the type of data to map[string]interface{}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data is not of type map[string]interface{}")
	}

	// Update the state in the dataMap
	updatedData := updateState(dataMap)
	return updatedData, nil
}

func updateState(data map[string]interface{}) map[string]interface{} {
	// Access the custom_fields map
	if customFields, ok := data["custom_fields"].(map[string]interface{}); ok {
		// Update the state from Reserved to Available
		if customFields["state"] == "Reserved" {
			customFields["state"] = "Available"
			customFields["session_id"] = ""
		}
	}
	return data
}

package inventory

// go clean -modcache
import (
	"encoding/json"
	"fmt"
	"keysight/laas/controller/config"
	"keysight/laas/controller/internal/profile"
	"keysight/laas/controller/internal/utils"
	"os"
	"strings"
	"time"
	// "github.com/openconfig/ondatra/gnmi/oc/platform"
)

type Interface struct {
	Name   string `json:"name"`
	Speed  int    `json:"speed"`
	Status string `json:"status"`
	// Add more fields as needed
}
type Devicecount struct {
	Devices map[string]Device `json:"devices"`
}

// Device represents the structure of a device
type Device struct {
	// ID         float64       `json:"id"`
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Vendor     string                 `json:"vendor"`
	Model      string                 `json:"model"`
	Role       string                 `json:"role"`
	Platform   string                 `json:"platform"`
	Image      string                 `json:"image"`
	Attributes map[string]interface{} `json:"attributes"`
	Handles    []Handles              `json:"handles"`
	Ports      []interface{}          `json:"ports"`
	// Add more fields as needed
}

// Inventory represents the inventory details of a device
type Attributes struct {
	// Add more fields as needed
	Type     string `json:"devicetype"`
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Handles struct {
	Connection string `json:"connection"`
	Credential string `json:"credential"`
	Name       string `json:"name"`
	Via        string `json:"via"`
	// Add more fields as needed
}

// Dut represents the structure of the device under "duts" key
type Dut struct {
	Name    string            `json:"desc"`
	Devices map[string]Device `json:"devices"`
	Links   []DutLink         `json:"links"`
}

type DutLink struct {
	Src DutLinkEndpoint `json:"src"`
	Dst DutLinkEndpoint `json:"dst"`
}

type DutLinkEndpoint struct {
	Device string `json:"device"`
	Port   string `json:"port"`
}

// Counter represents a simple counter for generating auto-increment IDs
type Counter struct {
	Value int
}

func (c *Counter) nextID() int {
	c.Value++
	return c.Value
}

var log = config.GetLogger("inventory")

// AddDevice adds a new device with interfaces and an auto-incrementing ID to the provided map
func AddDevice(counter *Counter, devices map[int]Device, name, role, deviceType, platform, image, state string, manufacturer string, connection string, credential string, handlename string, via string, inventoryAttrs map[string]interface{}, interfaces []interface{}) map[int]Device {
	deviceID := counter.nextID()
	fmt.Println("Mohan inventoryAttrs:", inventoryAttrs)
	fmt.Println("Mohan interfaces:", interfaces)
	devices[deviceID] = Device{
		ID:         name,
		Name:       strings.ToLower(name),
		Vendor:     strings.ToLower(manufacturer),
		Model:      strings.ToLower(deviceType),
		Role:       role,
		Platform:   strings.ToLower(platform),
		Image:      strings.ToLower(image),
		Attributes: inventoryAttrs,
		Handles: []Handles{
			{
				Connection: connection,
				Credential: credential,
				Name:       handlename,
				Via:        via,
				// Add more fields as needed
			},
			// Add more Handles as needed
		},
		Ports: interfaces,
	}
	return devices
}

func createInventory(listOfDicts []map[string]interface{}, linksOfDicts []map[string]interface{}, inventoryFile string, inventoryType string) {
	defer profile.LogFuncDuration(time.Now(), "createInventory", "", "inventory")
	// Parse JSON output into Go data structure
	// Initialize an empty map for devices
	devices := make(map[int]Device)
	devicesSlice := make(map[string]Device)
	for _, dict := range listOfDicts {
		// Get values using keys
		name := dict["Name"].(string)
		role := dict["Role"].(string)
		deviceType := dict["DeviceType"].(string)
		// manufacturer := dict["Manufacturer"].(string)
		manufacturer := dict["Vendor"].(string)
		platform := dict["Platform"].(string)
		image := dict["Image"].(string)
		state := dict["State"].(string)
		connection := dict["Connec"].(string)
		credential := dict["Cred"].(string)
		handlename := dict["HandleName"].(string)
		via := dict["Via"].(string)
		inventoryAttrs := dict["inventoryAttr"].(map[string]interface{})
		interfaces := dict["interfaces"].([]interface{})
		idCounter := &Counter{}
		if strings.ToLower(inventoryType) == "all" {
			devices = AddDevice(idCounter, devices, name, role, deviceType, platform, image, state, manufacturer, connection, credential, handlename, via, inventoryAttrs, interfaces)
		} else {
			if strings.ToLower(state) != "reserved" {
				devices = AddDevice(idCounter, devices, name, role, deviceType, platform, image, state, manufacturer, connection, credential, handlename, via, inventoryAttrs, interfaces)
			} else {
				devices = make(map[int]Device)
			}
		}
		// Convert the map to a slice
		if strings.ToLower(role) == "dut" || strings.ToLower(role) == "ate" || strings.ToLower(role) == "tgen" || strings.ToLower(role) == "l1s" {
			for _, device := range devices {
				devicesSlice[device.ID] = device
			}
		}
	}
	// Initialize an empty slice for links
	links := make([]DutLink, 0)

	// Iterate through linksOfDicts and convert them to DutLink instances
	for _, linkDict := range linksOfDicts {
		src := linkDict["src"].(string)
		dst := linkDict["dst"].(string)
		srcDevice, srcPort := utils.SplitString(src)
		dstDevice, dstPort := utils.SplitString(dst)

		// Create DutLinkEndpoint instances for Src and Dst
		srcEndpoint := DutLinkEndpoint{
			Device: srcDevice,
			Port:   srcPort,
		}
		dstEndpoint := DutLinkEndpoint{
			Device: dstDevice,
			Port:   dstPort,
		}
		// Create DutLink instance and append it to the links slice
		link := DutLink{
			Src: srcEndpoint,
			Dst: dstEndpoint,
		}
		links = append(links, link)
	}

	// Create Dut with devices and links
	duts := Dut{
		Name:    "Inventory",
		Devices: devicesSlice,
		Links:   links,
	}
	// Marshal the Dut into JSON
	dutsJSON, err := json.MarshalIndent(duts, "", "    ")
	if err != nil {
		log.Fatal().Msgf("Failed to marshal duts: %v", err)
		return
	}
	// Write JSON to a file
	err = os.WriteFile(inventoryFile, dutsJSON, 0644)
	if err != nil {
		log.Fatal().Msgf("Failed to write Dut JSON: %v", err)
		return
	}
	log.Info().Msgf("JSON written to %v", inventoryFile)
	log.Debug().Interface("Inventory data", dutsJSON).Msg("Obtained inventory data")
}

// FileExists checks if a file exists
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("file does not exist: %s", filePath)
	}
	return true, nil
}

func UpdateInventory(netboxApiURL string, netboxApiToken string, userID string, devices map[string][]map[string]interface{}) (string, error) {
	filePath := "output.json"
	exists, err := FileExists(filePath)
	if err != nil {
		// log.Fatal().Msgf("Failed to read file: %v, Error: %v", filePath, err) // Log the error and terminate the program
		return "", fmt.Errorf("failed to read file: %v, Error: %v", filePath, err)
	}
	if exists {
		updateerr := updateDevicesData(filePath, netboxApiURL, netboxApiToken, userID, devices)
		if updateerr != nil {
			// log.Fatal().Msgf("updateDevicesData failed: %v", updateerr)
			return "", fmt.Errorf("%v", updateerr)
		} else {
			log.Info().Msg("Node/Interfaces details updated successfully as per testbed details.")
			return "Node/Interfaces details updated successfully as per testbed details.", nil
		}
	} else {
		// log.Fatal().Msgf("Output file does not exist.")
		return "", fmt.Errorf("output file does not exist: %v", filePath)
	}
}

func GetCreateInvFromNetbox(netboxApiURL string, netboxApiToken string) {
	defer profile.LogFuncDuration(time.Now(), "GetCreateInvFromNetbox", "", "inventory")

	output := getDevicesData(netboxApiURL, netboxApiToken)
	var listOfDicts []map[string]interface{}
	err := json.Unmarshal(output, &listOfDicts)
	if err != nil {
		log.Fatal().Msgf("Failed to unmarsh the JSON file: %v", err)
		return
	}
	linksoutput := getDevicesLinks(netboxApiURL, netboxApiToken)
	var linksOfDicts []map[string]interface{}
	err = json.Unmarshal(linksoutput, &linksOfDicts)
	if err != nil {
		log.Fatal().Msgf("Failed to unmarsh the JSON file: %v", err)
		return
	}
	createInventory(listOfDicts, linksOfDicts, "inventory_global.json", "all")
	createInventory(listOfDicts, linksOfDicts, "inventory.json", "NA")
}

func ReleaseStateWithInvenData(releaseState map[string][]map[string]interface{}, user_id string) error {
	updateerr := updateNodeState(releaseState, user_id)
	if updateerr != nil {
		return fmt.Errorf("%v", updateerr)
	}
	return nil
}

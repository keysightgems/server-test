package cafy

import (
	"encoding/json"
	"fmt"
	"keysight/laas/controller/internal/profile"
	"os"
	"strconv"
	"strings"
	"time"
)

type Connection struct {
	Device string `json:"device"`
	Port   string `json:"port"`
}

// Link represents a pair of connections
type LinkConn struct {
	Src Connection `json:"src"`
	Dst Connection `json:"dst"`
}

type Output struct {
	Desc    string               `json:"desc"`
	Devices map[string]DeviceOut `json:"devices"`
	Links   []LinkOut            `json:"links"`
}

type DeviceOut struct {
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
	Ports      map[string]PortOut     `json:"ports"`
	Handles    interface{}            `json:"handles"`
}

type PortOut struct {
	ID          string                 `json:"Id"`
	Name        string                 `json:"name"`
	Speed       string                 `json:"speed"`
	Pmd         string                 `json:"pmd"`
	Transceiver string                 `json:"transceiver"`
	Attributes  map[string]interface{} `json:"attributes"`
}

type LinkOut struct {
	Dst EndpointOut `json:"dst"`
	Src EndpointOut `json:"src"`
}

type EndpointOut struct {
	Device string `json:"device"`
	Port   string `json:"port"`
}

func generateStructLink(data string) []LinkConn {
	// Clean up the input data
	data = strings.Trim(data, "[]")

	// Split the data into pairs
	pairs := strings.Split(data, "}} {{")

	// Initialize a slice to hold the links
	var links []LinkConn

	// Parse each pair
	for _, pair := range pairs {
		// Remove the outer curly braces
		pair = strings.Trim(pair, "{}")

		// Split the pair into two connections
		connections := strings.Split(pair, "} {")
		if len(connections) != 2 {
			log.Info().Interface("Invalid pair format:", pair).Msg("generateStructLink")
			continue
		}

		// Parse the source connection
		srcParts := strings.Split(connections[0], " ")
		if len(srcParts) != 2 {
			log.Info().Interface("Invalid source format:", connections[0]).Msg("generateStructLink")
			continue
		}
		src := Connection{
			Device: srcParts[0],
			Port:   srcParts[1],
		}

		// Parse the destination connection
		dstParts := strings.Split(connections[1], " ")
		if len(dstParts) != 2 {
			log.Info().Interface("Invalid destination format:", connections[1]).Msg("generateStructLink")
			continue
		}
		dst := Connection{
			Device: dstParts[0],
			Port:   dstParts[1],
		}

		// Create a link and add it to the slice
		link := LinkConn{
			Src: src,
			Dst: dst,
		}
		links = append(links, link)
	}
	return links
}
func GenerateInterfaceMap(links []LinkConn, deviceId string, role string) map[string]map[string]string {
	defer profile.LogFuncDuration(time.Now(), "GenerateInterfaceMap", "", "cafy")
	interfaceMap := make(map[string]map[string]string)
	linkCounts := make(map[string]int)

	for _, linkPair := range links {
		src := linkPair.Src
		dst := linkPair.Dst

		if interfaceMap[src.Device] == nil {
			interfaceMap[src.Device] = make(map[string]string)
		}
		if interfaceMap[dst.Device] == nil {
			interfaceMap[dst.Device] = make(map[string]string)
		}

		// Create a consistent key for the link between two devices
		var linkKey string
		linkKey = fmt.Sprintf("%s_%s", src.Device, dst.Device)
		if src.Device > dst.Device {
			linkKey = fmt.Sprintf("%s_%s", dst.Device, src.Device)
		}

		// Increment the link count for the devices
		linkCounts[linkKey]++

		// Create the link name based on the role and link count
		var linkName string
		if src.Device > dst.Device {
			linkName = fmt.Sprintf("%s_%s_%d", dst.Device, src.Device, linkCounts[linkKey])
		} else {
			linkName = fmt.Sprintf("%s_%s_%d", src.Device, dst.Device, linkCounts[linkKey])
		}

		// Add the link to the interface map
		interfaceMap[src.Device][src.Port] = linkName
		interfaceMap[dst.Device][dst.Port] = linkName
	}

	// Filter interfaceMap based on deviceId
	filteredInterfaceMap := make(map[string]map[string]string)
	for device, ports := range interfaceMap {
		if device == deviceId || role == "ate" {
			filteredInterfaceMap[device] = ports
		} else {
			for port, linkName := range ports {
				linkParts := strings.Split(linkName, "_")
				if len(linkParts) == 3 && (linkParts[0] == deviceId || linkParts[1] == deviceId) {
					if filteredInterfaceMap[device] == nil {
						filteredInterfaceMap[device] = make(map[string]string)
					}
					filteredInterfaceMap[device][port] = linkName
				}
			}
		}
	}
	// Iterate over the map and delete entries not equal to deviceId
	for key := range filteredInterfaceMap {
		if key != deviceId {
			delete(filteredInterfaceMap, key)
		}
	}
	return filteredInterfaceMap
}

func updateOutputJson() {
	// Read the JSON file
	file, err := os.ReadFile("output.json")
	if err != nil {
		log.Fatal().Msgf("Error reading file: %s, error: %s", file, err.Error())
		return
	}

	// Parse the JSON data
	var data Output
	if err := json.Unmarshal(file, &data); err != nil {
		log.Fatal().Msgf("Error unmarshalling JSON: %s", err.Error())
		return
	}

	// Track the next IDs for DUTs and ATEs
	nextDutID := 1
	nextAteID := 1

	// Create a map to store the new IDs for devices
	deviceIDMap := make(map[string]string)

	// Update devices
	for key, device := range data.Devices {
		var newID string

		// Determine the new ID based on the role
		if strings.ToLower(device.Attributes["role"].(string)) == "dut" {
			newID = fmt.Sprintf("R%d", nextDutID)
			nextDutID++
		} else if strings.ToLower(device.Attributes["role"].(string)) == "ate" {
			newID = fmt.Sprintf("T%d", nextAteID)
			nextAteID++
		} else {
			newID = device.ID
		}

		// Store the new ID in the map
		deviceIDMap[device.ID] = newID

		// Update the device ID and name
		device.ID = newID
		device.Attributes["name"] = newID

		// Update ports Ids
		for portKey, port := range device.Ports {
			parts := strings.Split(port.ID, ":")
			if len(parts) > 1 {
				port.ID = fmt.Sprintf("%s:%s", newID, parts[1])
				device.Ports[portKey] = port
			}
		}

		// Replace the device in the map
		data.Devices[key] = device
	}

	// Update links
	for i, link := range data.Links {
		if newID, ok := deviceIDMap[link.Src.Device]; ok {
			data.Links[i].Src.Device = newID
		}
		if newID, ok := deviceIDMap[link.Dst.Device]; ok {
			data.Links[i].Dst.Device = newID
		}
	}

	// Write the updated data back to the JSON file
	updatedData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal().Msgf("Error marshalling updated data: %s", err.Error())
		return
	}

	if err := os.WriteFile("output.json", updatedData, 0644); err != nil {
		log.Fatal().Msgf("Error writing updated file: %s", err.Error())
		return
	}
	log.Info().Msg("Updated JSON file successfully with the proper router names.")
}

func stringConversion(value string, conType string) interface{} {
	if value == "null" {
		switch conType {
		case "int":
			return 0
		case "bool":
			return false
		case "string":
			return ""
		}
	}

	switch conType {
	case "int":
		conValue, err := strconv.Atoi(value)
		if err != nil {
			log.Error().Err(err).Msgf("Error converting value to integer: %s", err)
			return 0 // Return 0 for integer conversion errors
		}
		return conValue
	case "bool":
		conValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Error().Err(err).Msgf("Invalid boolean value: %s", value)
			return false // Return false for boolean conversion errors
		}
		return conValue
	case "string":
		return value
	}

	return value
}

func telnetConnection(node *Node, device Device) {
	telnetDefault := stringConversion(device.Attributes.TelnetDefault, "bool")
	telnetVia := stringConversion(device.Attributes.TelnetVia, "string").(string)
	handle := Handle{
		Connection: "telnet",
		Credential: "default",
		Name:       "telnet",
		Via:        telnetVia,
	}
	if telnetDefaultBool, ok := telnetDefault.(bool); ok && telnetDefaultBool {
		handle.DefaultHandle = telnetDefaultBool
	}
	node.Handles = append(node.Handles, handle)
}

func sshConnection(node *Node, device Device) {
	sshDefault := stringConversion(device.Attributes.SshDefault, "bool")
	sshVia := stringConversion(device.Attributes.SshVia, "string").(string)
	handle := Handle{
		Connection: "ssh",
		Credential: "default",
		Name:       "vty",
		Via:        sshVia,
	}
	if sshDefaultBool, ok := sshDefault.(bool); ok && sshDefaultBool {
		handle.DefaultHandle = sshDefaultBool
	}
	node.Handles = append(node.Handles, handle)
}

func haConnection(node *Node, device Device) {
	haDefault := stringConversion(device.Attributes.ConsoleDefault, "bool")
	haVia := stringConversion(device.Attributes.ConsoleVia, "string").(string)
	handle := Handle{
		Connection: "ha",
		Credential: "default",
		Name:       "console",
		Via:        haVia,
	}
	if haDefaultBool, ok := haDefault.(bool); ok && haDefaultBool {
		handle.DefaultHandle = haDefaultBool
	}
	node.Handles = append(node.Handles, handle)
}

func ydkConnection(node *Node, device Device) {
	ydkDefault := stringConversion(device.Attributes.YdkDefault, "bool")
	ydkVia := stringConversion(device.Attributes.YdkVia, "string").(string)
	handle := Handle{
		Connection: "ydk",
		Credential: "default",
		Name:       "ydk",
		Via:        ydkVia,
	}
	if ydkDefaultBool, ok := ydkDefault.(bool); ok && ydkDefaultBool {
		handle.DefaultHandle = ydkDefaultBool
	}
	node.Handles = append(node.Handles, handle)
}

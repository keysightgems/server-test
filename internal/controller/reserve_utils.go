package controller

import (
	"encoding/json"
	"fmt"
	"keysight/laas/controller/internal/profile"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/open-traffic-generator/openl1s/gol1s"
	"github.com/open-traffic-generator/opentestbed/goopentestbed"
	graph "github.com/openconfig/ondatra/binding/portgraph"
)

var InventoryConfig Inventory
var InventoryGraph graph.ConcreteGraph
var ConfigNodesToDevices map[*graph.ConcreteNode]Device
var ConfigPortsToPorts map[*graph.ConcretePort]Port

type Inventory struct {
	Desc    string            `json:"desc"`
	Devices map[string]Device `json:"devices"`
	Links   []Link            `json:"links"`
}

type Testbed struct {
	Desc    string             `json:"desc"`
	Devices map[string]BDevice `json:"devices"`
	Links   []Link             `json:"links"`
}

type Device struct {
	Id       string            `json:"id"`
	Model    string            `json:"model,omitempty"`
	Role     string            `json:"role,omitempty"`
	Vendor   string            `json:"vendor,omitempty"`
	Name     string            `json:"name,omitempty"`
	Platform string            `json:"platform,omitempty"`
	Image    string            `json:"image,omitempty"`
	Attrs    map[string]string `json:"attributes"`
	Services []Service         `json:"services"`
	Handles  []Handle          `json:"handles"`
	Ports    []Port            `json:"ports"`
}

type Service struct {
	Name          string `json:"name"`
	AddressFamily string `json:"address_family"`
	Address       string `json:"address"`
	Protocol      string `json:"protocol"`
	Port          int    `json:"port"`
}

type Port struct {
	Id          string `json:"Id"`
	Name        string `json:"name"`
	Speed       string `json:"speed"`
	Pmd         string `json:"pmd"`
	Transceiver string `json:"transceiver"`

	Attrs map[string]string `json:"attributes"`
}

type Link struct {
	Dst InputLinkEndpoint `json:"dst"`
	Src InputLinkEndpoint `json:"src"`
}

type BDevice struct {
	Id       string            `json:"id"`
	Model    string            `json:"model,omitempty"`
	Role     string            `json:"role,omitempty"`
	Vendor   string            `json:"vendor,omitempty"`
	Name     string            `json:"name,omitempty"`
	Platform string            `json:"platform,omitempty"`
	Image    string            `json:"image,omitempty"`
	Attrs    map[string]string `json:"attributes"`
	Ports    map[string]Port   `json:"ports"`
	Handles  []Handle          `json:"handles"`
}

type InputAttributes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type InputInterface struct {
	Attributes  []InputAttributes `json:"attributes,omitempty"`
	Id          string            `json:"id"`
	Speed       string            `json:"speed,omitempty"`
	Name        string            `json:"name"`
	Pmd         string            `json:"pmd"`
	Transceiver string            `json:"transceiver"`
}

type InputDevice struct {
	Ports      []InputInterface  `json:"ports"`
	Handles    []Handle          `json:"handles"`
	Model      string            `json:"model,omitempty"`
	Role       string            `json:"role,omitempty"`
	Id         string            `json:"id"`
	Vendor     string            `json:"vendor,omitempty"`
	Name       string            `json:"name,omitempty"`
	Platform   string            `json:"platform,omitempty"`
	Image      string            `json:"image,omitempty"`
	Attributes []InputAttributes `json:"attributes,omitempty"`
}

type InputLink struct {
	Dst InputLinkEndpoint `json:"dst"`
	Src InputLinkEndpoint `json:"src"`
}

type InputLinkEndpoint struct {
	Device string `json:"device"`
	Port   string `json:"port"`
}

type InputData struct {
	Devices []InputDevice `json:"devices"`
	Links   []InputLink   `json:"links"`
}

type Handle struct {
	Connection string `json:"connection"`
	Credential string `json:"credential"`
	Name       string `json:"name"`
	Via        string `json:"via"`
}

type L1Swport struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}
type TestData struct {
	Desc    string            `json:"desc"`
	Devices map[string]Device `json:"devices"`
	Links   []Link            `json:"links"`
}

func LoadConcreteGraph() {
	log.Info().Msg("Invoked LoadConcreteGraph")
	defer profile.LogFuncDuration(time.Now(), "LoadConcreteGraph", "", "controller")

	nodes := []*graph.ConcreteNode{}
	edges := []*graph.ConcreteEdge{}
	portPointers := map[string]*graph.ConcretePort{}

	for dname, device := range InventoryConfig.Devices {
		ports := []*graph.ConcretePort{}

		for _, port := range device.Ports {
			if port.Attrs == nil {
				port.Attrs = map[string]string{"reserved": "no"}
			} else {
				port.Attrs["speed"] = port.Speed
				port.Attrs["name"] = port.Name
				port.Attrs["pmd"] = port.Pmd
				port.Attrs["transceiver"] = strings.ToLower(port.Transceiver)
			}

			if strings.ToLower(port.Attrs["State"]) == "reserved" || strings.ToLower(port.Attrs["state"]) == "reserved" {
				port.Attrs["reserved"] = "yes"
			} else {
				port.Attrs["reserved"] = "no"
			}
			newPort := &graph.ConcretePort{Desc: (dname + ":" + port.Id), Attrs: port.Attrs}
			ports = append(ports, newPort)
			ConfigPortsToPorts[newPort] = port
			portPointers[dname+":"+port.Id] = newPort
		}

		if device.Attrs == nil {
			device.Attrs = map[string]string{}
		} else {
			device.Attrs["vendor"] = strings.ToLower(device.Vendor)
			device.Attrs["model"] = strings.ToLower(device.Model)
			device.Attrs["role"] = device.Role
			device.Attrs["name"] = strings.ToLower(device.Name)
			device.Attrs["platform"] = strings.ToLower(device.Platform)
			device.Attrs["image"] = strings.ToLower(device.Image)
		}
		if device.Handles == nil {
			device.Handles = []Handle{}
		}
		if strings.ToLower(device.Attrs["State"]) == "reserved" || strings.ToLower(device.Attrs["state"]) == "reserved" {
			device.Attrs["reserved"] = "yes"
		} else {
			device.Attrs["reserved"] = "no"
		}
		newNode := &graph.ConcreteNode{Desc: dname, Ports: ports, Attrs: device.Attrs}
		nodes = append(nodes, newNode)
		ConfigNodesToDevices[newNode] = device
	}

	InventoryGraph.Nodes = nodes

	for _, link := range InventoryConfig.Links {
		srcPort, srcExists := portPointers[link.Src.Device+":"+link.Src.Port]
		dstPort, dstExists := portPointers[link.Dst.Device+":"+link.Dst.Port]
		if !srcExists || !dstExists {
			// Handle missing port pointers
			continue
		}
		newEdge := &graph.ConcreteEdge{
			Src: srcPort,
			Dst: dstPort,
		}

		edges = append(edges, newEdge)
	}

	InventoryGraph.Edges = edges

	log.Info().Interface("InventoryGraph", InventoryGraph).Msg("Inventory graph")
}

func LoadAbstractGraph(testbedConfig Testbed, testbed *graph.AbstractGraph) {
	log.Info().Msg("Invoked LoadAbstractGraph")
	defer profile.LogFuncDuration(time.Now(), "LoadAbstractGraph", "", "controller")

	nodes := []*graph.AbstractNode{}
	edges := []*graph.AbstractEdge{}
	portPointers := map[string]*graph.AbstractPort{}

	for dname, device := range testbedConfig.Devices {
		ports := []*graph.AbstractPort{}

		for pid, port := range device.Ports {
			if port.Attrs == nil {
				port.Attrs = map[string]string{"reserved": "no"}
			}

			port.Attrs["reserved"] = "no"
			portConstraints := map[string]graph.PortConstraint{}

			for aid, attribute := range port.Attrs {
				portConstraints[aid] = graph.Equal(attribute)
			}

			newPort := &graph.AbstractPort{Desc: (dname + ":" + pid), Constraints: portConstraints}
			ports = append(ports, newPort)
			portPointers[dname+":"+pid] = newPort
		}

		if device.Attrs == nil {
			device.Attrs = map[string]string{}
		}

		device.Attrs["reserved"] = "no"
		deviceConstraints := map[string]graph.NodeConstraint{}

		for aid, attribute := range device.Attrs {
			deviceConstraints[aid] = graph.Equal(attribute)
		}

		newNode := &graph.AbstractNode{Desc: dname, Ports: ports, Constraints: deviceConstraints}
		nodes = append(nodes, newNode)
	}

	testbed.Nodes = nodes

	for _, link := range testbedConfig.Links {
		srcPort := portPointers[link.Src.Device+":"+link.Src.Port]
		dstPort := portPointers[link.Dst.Device+":"+link.Dst.Port]

		newEdge := &graph.AbstractEdge{
			Src: srcPort,
			Dst: dstPort,
		}

		edges = append(edges, newEdge)
	}

	testbed.Edges = edges

	log.Info().Interface("Testbed", testbed).Msg("Testbed Graph")
}

func ConvertData(srcData goopentestbed.Testbed) Testbed {
	log.Info().Msg("Invoked ConvertData")
	defer profile.LogFuncDuration(time.Now(), "ConvertData", "", "controller")

	destData := Testbed{
		Desc:    "testbed",
		Devices: make(map[string]BDevice),
	}
	deviceNameMap := make(map[string]string)
	for _, srcDevice := range srcData.Devices().Items() {
		destDevice := BDevice{
			Id:      srcDevice.Id(),
			Role:    string(srcDevice.Role()),
			Attrs:   make(map[string]string),
			Ports:   make(map[string]Port),
			Handles: make([]Handle, 0), // Initialize Handles slice
		}
		if srcDevice.HasVendor() {
			destDevice.Attrs["vendor"] = strings.ToLower(srcDevice.Vendor())
			destDevice.Vendor = strings.ToLower(srcDevice.Vendor())
		}
		if srcDevice.HasName() {
			destDevice.Attrs["name"] = strings.ToLower(srcDevice.Name())
			destDevice.Name = strings.ToLower(srcDevice.Name())
		}
		if srcDevice.HasModel() {
			destDevice.Attrs["model"] = strings.ToLower(srcDevice.Model())
			destDevice.Model = strings.ToLower(srcDevice.Model())
		}
		if srcDevice.Role() != "" {
			destDevice.Attrs["role"] = string(srcDevice.Role())
			destDevice.Role = string(srcDevice.Role())
		}
		if srcDevice.HasPlatform() {
			destDevice.Attrs["platform"] = strings.ToLower(srcDevice.Platform())
			destDevice.Platform = strings.ToLower(srcDevice.Platform())
		}
		if srcDevice.HasImage() {
			destDevice.Attrs["image"] = strings.ToLower(srcDevice.Image())
			destDevice.Image = strings.ToLower(srcDevice.Image())
		}
		// Process device attributes
		for _, deviceAttr := range srcDevice.Attributes().Items() {
			destDevice.Attrs[strings.ToLower(deviceAttr.Key())] = strings.ToLower(deviceAttr.Value())

		}
		// Process interfaces
		for _, srcInterface := range srcDevice.Ports().Items() {
			destPort := Port{
				Id:    srcInterface.Id(),
				Attrs: make(map[string]string),
			}

			// Process interface attributes
			for _, srcAttr := range srcInterface.Attributes().Items() {
				destPort.Attrs[strings.ToLower(srcAttr.Key())] = strings.ToLower(srcAttr.Value())
			}

			// Process speed attribute
			if srcInterface.Speed() != "" && srcInterface.Speed() != "SPEED_UNSPECIFIED" {
				destPort.Attrs["speed"] = string(srcInterface.Speed())
			}
			if srcInterface.HasName() {
				destPort.Attrs["name"] = srcInterface.Name()
			}
			if srcInterface.HasPmd() && string(srcInterface.Pmd()) != "PMD_UNSPECIFIED" {
				destPort.Attrs["pmd"] = string(srcInterface.Pmd())
			}
			if srcInterface.HasTransceiver() {
				destPort.Attrs["transceiver"] = strings.ToLower(srcInterface.Transceiver())
			}

			destDevice.Ports[srcInterface.Id()] = destPort
		}

		destData.Devices[srcDevice.Id()] = destDevice
		// Create a mapping of old device names to new ones
		deviceNameMap[srcDevice.Id()] = destDevice.Id
	}

	// Process links
	for _, srcLink := range srcData.Links().Items() {
		// Update device names in the links based on the mapping
		srcDeviceName := deviceNameMap[srcLink.Src().Device()]
		dstDeviceName := deviceNameMap[srcLink.Dst().Device()]
		srcPortName := srcLink.Src().Port()
		dstPortName := srcLink.Dst().Port()

		destLink := Link{
			Src: InputLinkEndpoint{
				Device: srcDeviceName,
				Port:   srcPortName,
			},
			Dst: InputLinkEndpoint{
				Device: dstDeviceName,
				Port:   dstPortName,
			},
		}
		destData.Links = append(destData.Links, destLink)
	}

	log.Debug().Interface("Testbed data", destData).Msg("Converted testbed data")
	return destData
}

func LoadTestbedData(filePath string) InputData {
	log.Info().Msg("Invoked LoadTestbedData")
	defer profile.LogFuncDuration(time.Now(), "LoadTestbedData", "", "controller")

	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal().Msgf("Failed to read file: %s, error: %s", filePath, err.Error())
	}
	var testbedData InputData
	err = json.Unmarshal(jsonData, &testbedData)
	if err != nil {
		log.Fatal().Msgf("Failed to unmarshal testbedData, error: %s", err.Error())
	}

	log.Debug().Interface("Testbed data", testbedData).Msg("Loaded testbed data")
	return testbedData
}

func LoadInventoryData(filePath string) Inventory {
	log.Info().Msg("Invoked LoadInventoryData")
	defer profile.LogFuncDuration(time.Now(), "LoadInventoryData", "", "controller")

	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal().Msgf("Failed to read file: %s, error: %s", filePath, err.Error())
	}
	var inventoryData Inventory
	err = json.Unmarshal(jsonData, &inventoryData)
	if err != nil {
		log.Fatal().Msgf("Failed to unmarshal inventoryData, error: %s", err.Error())
	}

	return inventoryData
}

func convertAttributesToStrings(data map[string]interface{}) {
	log.Info().Msg("Invoked convertAttributesToStrings")
	for _, device := range data["devices"].(map[string]interface{}) {
		attributes := device.(map[string]interface{})["attributes"].(map[string]interface{})
		for key, value := range attributes {
			switch v := value.(type) {
			case int:
				attributes[key] = strconv.Itoa(v)
			case float64:
				attributes[key] = strconv.FormatFloat(v, 'f', -1, 64)
			case bool:
				attributes[key] = strconv.FormatBool(v)
			}
		}
	}
}

func ConvertInventoryDataType() {
	log.Info().Msg("Invoked ConvertInventoryDataType")
	defer profile.LogFuncDuration(time.Now(), "ConvertInventoryDataType", "", "controller")

	// Read the JSON file
	data, err := os.ReadFile("inventory.json")
	if err != nil {
		log.Fatal().Msgf("Failed to read file: %v", err)
	}

	// Unmarshal the JSON data into a map[string]interface{}
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		log.Fatal().Msgf("Failed to unmarshal JSON: %v", err)
	}

	// Convert integer and boolean values in the attributes dictionary to strings
	convertAttributesToStrings(jsonData)

	// Marshal the modified JSON data back to a JSON string
	modifiedData, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatal().Msgf("Failed to marshal JSON: %v", err)
	}

	// Write the modified JSON to a new file
	err = os.WriteFile("inventory.json", modifiedData, 0644)
	if err != nil {
		log.Fatal().Msgf("Failed to write file: %v", err)
	}
}

func ProcessInventory(filePath string, destLink Link) (map[string]L1Swport, error) {
	log.Info().Msg("Invoked ProcessInventory to get the Switch connected Ports")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	var inventory Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	deviceMap := make(map[string]L1Swport)
	portPattern := regexp.MustCompile(`^\d+(\.\d+){1,3}$`)
	for _, link := range inventory.Links {
		if (link.Dst.Port == destLink.Dst.Port && link.Dst.Device == destLink.Dst.Device) || (link.Src.Port == destLink.Src.Port && link.Src.Device == destLink.Src.Device) {
			if len(link.Src.Port) > 0 && (strings.ToLower(link.Src.Port)[0] == 'p' || portPattern.MatchString(link.Src.Port)) {
				if ports, ok := deviceMap[link.Src.Device]; ok {
					ports.Src = link.Src.Port
					deviceMap[link.Src.Device] = ports
				} else {
					deviceMap[link.Src.Device] = L1Swport{Src: link.Src.Port}
				}
			}
			if len(link.Dst.Port) > 0 && (strings.ToLower(link.Dst.Port)[0] == 'p' || portPattern.MatchString(link.Dst.Port)) {
				if ports, ok := deviceMap[link.Dst.Device]; ok {
					ports.Dst = link.Dst.Port
					deviceMap[link.Dst.Device] = ports
				} else {
					deviceMap[link.Dst.Device] = L1Swport{Dst: link.Dst.Port}
				}
			}
		}
	}
	// Merge multiple port mappings into a single Swport entry for each device
	for device, ports := range deviceMap {
		deviceMap[device] = L1Swport{Src: ports.Src, Dst: ports.Dst}
	}
	return deviceMap, nil
}

func setupSwitchLinks(deviceMap map[string]L1Swport, switchLocation string, userId string) error {
	log.Info().Msg("Invoked setupSwitchLinks to configure Switch Ports")
	if len(deviceMap) > 1 {
		// Skip processing if deviceMap has multiple keys
		return nil
	}
	api := gol1s.NewApi()
	api.NewGrpcTransport().SetLocation(switchLocation)
	l1sConfig := gol1s.NewConfig()

	// Parse the deviceMap to create links
	for _, ports := range deviceMap {
		link := l1sConfig.Links().Add()
		link.SetSrc(ports.Src).SetDst(ports.Dst)
	}
	// Append l1sConfig to the global list
	globalL1sConfigs = append(globalL1sConfigs, l1sConfig)
	// Add entry to the map
	userConfigs[userId] = globalL1sConfigs
	_, err := api.SetConfig(l1sConfig)
	if err != nil {
		return fmt.Errorf("failed to configure switch ports: %w", err)
	}
	return nil
}

func removeSwitchLinks(switchLocation string, userId string) error {
	log.Info().Msg("Invoked removeSwitchLinks method")
	api := gol1s.NewApi()
	api.NewGrpcTransport().SetLocation(switchLocation)
	for key, value := range userConfigs {
		if key == userId {
			for _, config := range value {
				// Modify the operation field from CREATE to DELETE
				config.SetOperation(gol1s.ConfigOperation.DELETE)
				_, err := api.SetConfig(config)
				if err != nil {
					return fmt.Errorf("failed to delete configured switch ports: %w", err)
				}
			}
		}

	}
	globalL1sConfigs = []gol1s.Config{}
	return nil
}

// Generate an 8-byte unique user ID
func generateUserID() (string, error) {
	// Get the hostname of the device
	// hostname, err := os.Hostname()
	// if err != nil {
	// 	return "", fmt.Errorf("failed to get hostname: %v", err)
	// }
	hostname := "common"

	// Get the current timestamp in RFC3339Nano format
	timestamp := time.Now().Format(time.RFC3339Nano)

	// Combine the hostname, random user ID, and timestamp
	combinedID := fmt.Sprintf("%s-%s", hostname, timestamp)

	return combinedID, nil
}

// Helper function to check if the userId is present in releaseState
func userPresentInReleaseState(userId string, releaseState map[string][]map[string]interface{}) bool {
	for key := range releaseState {
		if key == userId {
			return true
		}
	}
	return false
}

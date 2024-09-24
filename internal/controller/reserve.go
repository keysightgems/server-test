package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"keysight/laas/controller/config"
	"keysight/laas/controller/internal/framework/cafy"
	"keysight/laas/controller/internal/framework/ondatra"
	inven "keysight/laas/controller/internal/inventory/netbox"
	"keysight/laas/controller/internal/profile"
	"keysight/laas/controller/internal/utils"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/open-traffic-generator/openl1s/gol1s"
	"github.com/open-traffic-generator/opentestbed/goopentestbed"
	graph "github.com/openconfig/ondatra/binding/portgraph"
)

var log = config.GetLogger("controller")

var (
	releaseState     = make(map[string][]map[string]interface{})
	globalL1sConfigs []gol1s.Config
	userConfigs      = make(map[string][]gol1s.Config)
	apiMutex         sync.Mutex
)

func Reserve(data goopentestbed.Testbed) (goopentestbed.ReserveResponse, error) {
	defer profile.LogFuncDuration(time.Now(), "Reserve", "", "controller")
	// Lock the mutex before making the API call
	apiMutex.Lock()
	defer apiMutex.Unlock()
	// Clear the globalL1sConfigs at the start of the function
	globalL1sConfigs = []gol1s.Config{}

	// Generate a unique userID
	userID, usrerr := generateUserID()
	if usrerr != nil {
		return goopentestbed.NewReserveResponse(), fmt.Errorf("error generating user ID: %w", usrerr)
	}

	log.Info().Str("UserID", userID).Interface("Request", data).Msg("Reserve request")
	log.Debug().Str("UserID", userID).Interface("Request-Debug", data).Msg("Reserve request")
	log.Trace().Str("UserID", userID).Interface("Request-Trace", data).Msg("Reserve request")

	// Check for duplicate IDs in the data
	if _, err := CheckForDuplicateIDs(data); err != nil {
		return goopentestbed.NewReserveResponse(), err
	}

	var err error
	// Loading and processing testbed data
	testbedConfig := ConvertData(data)

	// Create Abstract Graphs
	testbed := graph.AbstractGraph{}
	LoadAbstractGraph(testbedConfig, &testbed)

	// Get inventory
	inven.GetCreateInvFromNetbox(*config.Config.NetboxApiURL, *config.Config.NetboxUserToken)

	// This line is for debugging purpose only
	// testbedData := LoadTestbedData("testbed.json")
	// Convert Inventory Data type from other types to string
	ConvertInventoryDataType()
	// Create Concrete Graph
	InventoryConfig = LoadInventoryData("inventory.json")
	InventoryGraph = graph.ConcreteGraph{}
	ConfigNodesToDevices = map[*graph.ConcreteNode]Device{}
	ConfigPortsToPorts = map[*graph.ConcretePort]Port{}
	LoadConcreteGraph()

	inventory := InventoryGraph
	// Print &testbed as JSON
	testbedJSON, err := json.MarshalIndent(&testbed, "", "  ")
	if err != nil {
		return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to marshal testbed to JSON: %w", err)
	}
	log.Info().RawJSON("Testbed", testbedJSON).Msg("Abstract Graph")

	// Print &inventory as JSON
	inventoryJSON, err := json.MarshalIndent(&inventory, "", "  ")
	if err != nil {
		return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to marshal inventory to JSON: %w", err)
	}
	log.Info().RawJSON("Inventory", inventoryJSON).Msg("Concrete Graph")
	assignment, err := graph.Solve(context.Background(), &testbed, &inventory)
	if err != nil {
		if strings.Contains(err.Error(), "edges") {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("found inventory mismatch: %s", "failed to find nodes/links in inventory, please correct the inventory configuration and try again.")
		} else {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("found inventory mismatch: %w", err)
		}
	}

	// Ensure map initialization
	if assignment.Port2Port == nil {
		assignment.Port2Port = make(map[*graph.AbstractPort]*graph.ConcretePort)
	}

	devices := map[string]BDevice{}
	for _, node := range testbed.Nodes {
		ports := map[string]Port{}
		for _, port := range node.Ports {
			ConfigPortsToPorts[assignment.Port2Port[port]].Attrs["reserved"] = "yes"
			newPort := Port{Id: assignment.Port2Port[port].Desc, Attrs: assignment.Port2Port[port].Attrs}
			ports[port.Desc] = newPort
		}
		ConfigNodesToDevices[assignment.Node2Node[node]].Attrs["reserved"] = "yes"
		newNode := BDevice{Id: assignment.Node2Node[node].Desc, Attrs: assignment.Node2Node[node].Attrs, Ports: ports}
		devices[node.Desc] = newNode
	}
	links := []Link{}
	for _, edge := range testbed.Edges {
		srcDevice, srcPort := utils.SplitString(assignment.Port2Port[edge.Src].Desc)
		dstDevice, dstPort := utils.SplitString(assignment.Port2Port[edge.Dst].Desc)
		srcEndpoint := InputLinkEndpoint{
			Device: srcDevice,
			Port:   srcPort,
		}
		dstEndpoint := InputLinkEndpoint{
			Device: dstDevice,
			Port:   dstPort,
		}
		destLink := Link{
			Src: srcEndpoint,
			Dst: dstEndpoint,
		}

		deviceMap, err := ProcessInventory("inventory.json", destLink)
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("%w", err)
		}

		if len(deviceMap) != 0 {
			err = setupSwitchLinks(deviceMap, *config.Config.L1SwitchLocation, userID)
			if err != nil {
				return goopentestbed.NewReserveResponse(), fmt.Errorf("%w", err)
			}
		}
		links = append(links, destLink)
	}
	content, err := json.Marshal(Testbed{Devices: devices, Links: links})
	if err != nil {
		return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to marshal data: %w", err)
	}

	err = os.WriteFile("output.json", content, 0644)
	if err != nil {
		return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to write data into file: %w", err)
	}
	msg, updateerr := inven.UpdateInventory(*config.Config.NetboxApiURL, *config.Config.NetboxUserToken, userID, releaseState)
	if updateerr != nil {
		// log.Fatal().Msgf("updateDevicesData failed: %v", updateerr)
		return goopentestbed.NewReserveResponse(), fmt.Errorf("updateDevicesData failed: %v", updateerr)
	}
	log.Info().Interface("UpdateInventory", msg).Msg("Update Inventory")
	var response string
	frameworkName := strings.ToLower(*config.Config.FrameworkName)
	switch frameworkName {
	case "cafy":
		// cafy.CafyMain()
		response, err = cafy.CafyMain()
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("error in CafyMain function: %w", err)
		}
	case "ondatra":
		// ondatra.OndatraMain()
		// bindingContent, err := ondatra.OndatraMain()
		// if err != nil {
		// 	return fmt.Errorf("error in OndatraMain: %w", err)
		// }
		// // Return JSON data
		// c.IndentedJSON(http.StatusCreated, bindingContent)
		// return nil
		response, err = ondatra.OndatraMain()
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("error in Ondatra function: %w", err)
		}
	default: //generic
		fileContent, err := os.ReadFile("output.json")
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to read output file: %w", err)
		}
		var outputData Testbed
		if err := json.Unmarshal(fileContent, &outputData); err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to unmarshal output data: %w", err)
		}
		resultJSON, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to convert data to JSON file: %w", err)
		}
		outputFilePath := "output.json"

		// Write the result JSON to the output file
		err = os.WriteFile(outputFilePath, resultJSON, 0644)
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to write data to output file: %w", err)
		}
		log.Info().Msg("Successfully generated generic testbed file")
		fileContent, err = os.ReadFile(outputFilePath)
		if err != nil {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("failed to read output file: %w", err)
		}

		response = string(fileContent)
		// c.Data(http.StatusOK, "application/json; charset=utf-8", fileContent)
	}

	log.Info().Str("UserID", userID).Interface("Response", response).Msg("Reserve response")
	result := goopentestbed.NewReserveResponse()
	result.YieldResponse().SetSessionid(userID)
	result.YieldResponse().SetTestbed(response)
	return result, nil
}

func Release(userId goopentestbed.Session) (goopentestbed.ReleaseResponse, error) {
	// Lock the mutex before making the API call
	apiMutex.Lock()
	defer apiMutex.Unlock()
	if len(releaseState) != 0 && userPresentInReleaseState(userId.Id(), releaseState) {
		configErr := removeSwitchLinks(*config.Config.L1SwitchLocation, userId.Id())
		if configErr != nil {
			return goopentestbed.NewReleaseResponse(), fmt.Errorf("%w", configErr)
		}
		nodeerr := inven.ReleaseStateWithInvenData(releaseState, userId.Id())
		if nodeerr != nil {
			return goopentestbed.NewReleaseResponse(), fmt.Errorf("%v", nodeerr)
		} else {
			result := goopentestbed.NewReleaseResponse()
			result.Warning().SetWarnings([]string{"Node/Interfaces details updated successfully as per testbed details."})
			return result, nil
		}
	} else {
		configErr := removeSwitchLinks(*config.Config.L1SwitchLocation, userId.Id())
		if configErr != nil {
			return goopentestbed.NewReleaseResponse(), fmt.Errorf("%w", configErr)
		}
		msg, nodeerr := inven.UpdateInventory(*config.Config.NetboxApiURL, *config.Config.NetboxUserToken, userId.Id(), releaseState)
		if nodeerr != nil {
			return goopentestbed.NewReleaseResponse(), fmt.Errorf("%v", nodeerr)
		} else {
			result := goopentestbed.NewReleaseResponse()
			result.Warning().SetWarnings([]string{msg})
			return result, nil
		}
	}

}

func CheckForDuplicateIDs(data goopentestbed.Testbed) (goopentestbed.ReserveResponse, error) {
	idSet := make(map[string]struct{})
	portIDSet := make(map[string]struct{})

	for _, node := range data.Devices().Items() {
		// Check if the node has a "state" attribute (case-insensitive)
		for _, attr := range node.Attributes().Items() {
			if strings.EqualFold(strings.ToLower(attr.Key()), "state") {
				return goopentestbed.NewReserveResponse(), fmt.Errorf("%s attribute not allowed for Device as user input", attr.Key())
			}
		}

		if _, exists := idSet[node.Id()]; exists {
			return goopentestbed.NewReserveResponse(), fmt.Errorf("duplicate device ID found in Input Data: %s", node.Id())
		}
		idSet[node.Id()] = struct{}{}

		// Check for duplicate port IDs within the device
		for _, port := range node.Ports().Items() {
			// Check if the port has a "state" attribute (case-insensitive)
			for _, attr := range port.Attributes().Items() {
				if strings.EqualFold(strings.ToLower(attr.Key()), "state") {
					return goopentestbed.NewReserveResponse(), fmt.Errorf("%s attribute not allowed for Port as user input", attr.Key())
				}
			}

			if _, exists := portIDSet[port.Id()]; exists {
				return goopentestbed.NewReserveResponse(), fmt.Errorf("duplicate port ID found in Input Data: %s", port.Id())
			}
			portIDSet[port.Id()] = struct{}{}
		}
	}

	return goopentestbed.NewReserveResponse(), nil
}

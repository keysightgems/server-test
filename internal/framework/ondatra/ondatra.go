package ondatra

import (
	"encoding/json"
	"fmt"
	"keysight/laas/controller/config"
	"keysight/laas/controller/internal/profile"
	"os"
	"strconv"
	"strings"
	"time"

	// "github.com/golang/protobuf/proto"
	bindpb "github.com/openconfig/featureprofiles/topologies/proto/binding"
	"google.golang.org/protobuf/encoding/prototext"
)

type Attributes struct {
	Address              string      `json:"address"`
	Connection           string      `json:"connection"`
	Credential           string      `json:"credential"`
	DeviceType           string      `json:"devicetype"`
	HandleName           string      `json:"handle_name"`
	Image                string      `json:"image"`
	Model                string      `json:"model"`
	Name                 string      `json:"name"`
	Password             string      `json:"password"`
	Platform             string      `json:"platform"`
	Reserved             string      `json:"reserved"`
	State                string      `json:"state"`
	Username             string      `json:"username"`
	Vendor               string      `json:"vendor"`
	Via                  string      `json:"via"`
	OptionsInsecure      interface{} `json:"options_insecure"`
	DutHostname          string      `json:"dut_hostname"`
	DutPort              interface{} `json:"dut_port"`
	GnmiDutPort          interface{} `json:"gnmi_dut_port"`
	GnmiDutTarget        string      `json:"gnmi_dut_target"`
	GnoiTarget           string      `json:"gnoi_target"`
	GnoiMaxRecvMsgSize   interface{} `json:"gnoi_max_recvmsgsize"`
	GnoiPort             interface{} `json:"gnoi_port"`
	AteHostname          string      `json:"ate_hostname"`
	AtePort              interface{} `json:"ate_port"`
	GnmiAtePort          interface{} `json:"gnmi_ate_port"`
	GnmiAteTarget        string      `json:"gnmi_ate_target"`
	OtgTarget            string      `json:"otg_target"`
	OtgPort              interface{} `json:"otg_port"`
	OtgInsecure          interface{} `json:"otg_insecure"`
	OtgTimeout           interface{} `json:"otg_timeout"`
	GnmiSkipVerify       interface{} `json:"gnmi_skipverify"`
	GnmiTimeout          interface{} `json:"gnmi_timeout"`
	ConfigCli            string      `json:"config_cli"`
	ConfigGribiFlush     interface{} `json:"config_gribiflush"`
	DutOptionsUser       string      `json:"dut_options_user"`
	DutOptionsPass       string      `json:"dut_options_pass"`
	DutOptionsSkipVerify interface{} `json:"dut_options_skipverify"`
	GribiTarget          string      `json:"gribi_target"`
	GribiPort            interface{} `json:"gribi_port"`
	P4rtTarget           string      `json:"p4rt_target"`
	P4rtPort             interface{} `json:"p4rt_port"`
	SshTarget            string      `json:"ssh_target"`
	SshPort              interface{} `json:"ssh_port"`
	SshUser              string      `json:"ssh_user"`
	SshPass              string      `json:"ssh_pass"`
}

type PortAttributes struct {
	Pmd         string `json:"pmd"`
	Name        string `json:"name"`
	Reserved    string `json:"reserved"`
	Speed       string `json:"speed"`
	Transceiver string `json:"transceiver"`
}

type Port struct {
	ID          string         `json:"Id"`
	Name        string         `json:"name"`
	Speed       string         `json:"speed"`
	Pmd         string         `json:"pmd"`
	Transceiver string         `json:"transceiver"`
	Attributes  PortAttributes `json:"attributes"`
}

type Device struct {
	ID         string          `json:"id"`
	Attributes Attributes      `json:"attributes"`
	Ports      map[string]Port `json:"ports"`
	Handles    interface{}     `json:"handles"`
}

type SrcDst struct {
	Device string `json:"device"`
	Port   string `json:"port"`
}

type Link struct {
	Dst SrcDst `json:"dst"`
	Src SrcDst `json:"src"`
}

type TestData struct {
	Desc    string            `json:"desc"`
	Devices map[string]Device `json:"devices"`
	Links   []Link            `json:"links"`
}

var log = config.GetLogger("ondatra")

func generateBindingContent(data TestData) *bindpb.Binding {
	log.Info().Msg("Invoked generateBindingContent")
	defer profile.LogFuncDuration(time.Now(), "generateBindingContent", "", "ondatra")

	bindingData := &bindpb.Binding{}
	var optionsPrinted bool
	// Map to store port IDs for each device
	portIDs := make(map[string]int)
	// Iterate over devices
	for _, device := range data.Devices {
		portCounter := 1
		if !optionsPrinted {
			// Set options
			bindingData.Options = &bindpb.Options{
				Username: device.Attributes.Username,
				Password: device.Attributes.Password,
			}
			optionsPrinted = true
		}
		deviceData := &bindpb.Device{
			Id: device.Attributes.Name,
		}

		// Add options for DUTs
		if device.Attributes.DeviceType == "DUT" {
			if !isNull(device.Attributes.DutPort) {
				deviceData.Name = checkTypeAndReturn(device.Attributes.DutHostname, device.Attributes.DutPort)
			}
			if !isNull(device.Attributes.OptionsInsecure) {
				options := &bindpb.Options{}
				deviceData.Options = deviceOptions(options, device)
			}

			if !isNull(device.Attributes.SshUser) || !isNull(device.Attributes.SshPass) && (!isNull(device.Attributes.SshTarget) && !isNull(device.Attributes.SshPort)) {
				deviceData.Ssh = &bindpb.Options{
					Target:   checkTypeAndReturn(device.Attributes.SshTarget, device.Attributes.SshPort),
					Username: device.Attributes.SshUser,
					Password: device.Attributes.SshPass,
				}
			}
			if device.Attributes.ConfigCli != "" {
				// Convert the string to []byte
				configBytes := []byte(device.Attributes.ConfigCli)

				// Create a slice of [][]byte with a single element
				configSlice := [][]byte{configBytes}

				// Assign the slice to the Config field
				deviceData.Config = &bindpb.Configs{
					Cli:        configSlice,
					GribiFlush: getBoolValue(device.Attributes.ConfigGribiFlush),
				}
			}
			if !isNull(device.Attributes.GnmiDutTarget) && !isNull(device.Attributes.GnmiDutPort) {
				deviceData.Gnmi = &bindpb.Options{
					Target: checkTypeAndReturn(device.Attributes.GnmiDutTarget, device.Attributes.GnmiDutPort),
				}
			}
			if !isNull(device.Attributes.GnoiTarget) && !isNull(device.Attributes.GnoiPort) && !isNull(device.Attributes.GnoiMaxRecvMsgSize) {
				gnoiMaxRecvMsgSize := int32(device.Attributes.GnoiMaxRecvMsgSize.(float64))
				deviceData.Gnoi = &bindpb.Options{
					Target:         checkTypeAndReturn(device.Attributes.GnoiTarget, device.Attributes.GnoiPort),
					MaxRecvMsgSize: gnoiMaxRecvMsgSize,
				}
			}
			if !isNull(device.Attributes.GribiTarget) && !isNull(device.Attributes.GribiPort) {
				deviceData.Gribi = &bindpb.Options{
					Target: checkTypeAndReturn(device.Attributes.GribiTarget, device.Attributes.GribiPort),
				}
			}
			if !isNull(device.Attributes.P4rtTarget) && !isNull(device.Attributes.P4rtPort) {
				deviceData.P4Rt = &bindpb.Options{
					Target: checkTypeAndReturn(device.Attributes.P4rtTarget, device.Attributes.P4rtPort),
				}
			}
			// Add ports based on links connected to the DUT
			for _, link := range data.Links {
				if link.Dst.Device == deviceData.Id {
					portID := "port" + strconv.Itoa(portCounter)
					portIDs[link.Dst.Port] = portCounter
					p := &bindpb.Port{
						Id:   portID,
						Name: link.Dst.Port,
					}
					deviceData.Ports = append(deviceData.Ports, p)
					portCounter++
				} else {
					portID := "port" + strconv.Itoa(portCounter)
					portIDs[link.Src.Port] = portCounter
					p := &bindpb.Port{
						Id:   portID,
						Name: link.Src.Port,
					}
					deviceData.Ports = append(deviceData.Ports, p)
					portCounter++
				}
			}
		}

		// Add options for ATEs
		if device.Attributes.DeviceType == "ATE" {
			if !isNull(device.Attributes.AtePort) && !isNull(device.Attributes.AteHostname) {
				deviceData.Name = checkTypeAndReturn(device.Attributes.AteHostname, device.Attributes.AtePort)
			}
			if !isNull(device.Attributes.OtgPort) && !isNull(device.Attributes.OtgInsecure) {
				otgTimeOut := int32(device.Attributes.OtgTimeout.(float64))
				deviceData.Otg = &bindpb.Options{
					Target:   checkTypeAndReturn(device.Attributes.OtgTarget, device.Attributes.OtgPort),
					Insecure: getBoolValue(device.Attributes.OtgInsecure),
					Timeout:  otgTimeOut,
				}
				if ok := deviceData.Otg.Insecure; !ok {
					deviceData.Otg.Insecure = false
				}
			}
			if !isNull(device.Attributes.GnmiAtePort) && !isNull(device.Attributes.GnmiSkipVerify) {
				gnmiTimeOut := int32(device.Attributes.GnmiTimeout.(float64))
				deviceData.Gnmi = &bindpb.Options{
					Target:     checkTypeAndReturn(device.Attributes.GnmiAteTarget, device.Attributes.GnmiAtePort),
					SkipVerify: getBoolValue(device.Attributes.GnmiSkipVerify),
					Timeout:    gnmiTimeOut,
				}
				if ok := deviceData.Gnmi.SkipVerify; !ok {
					deviceData.Gnmi.SkipVerify = false
				}
			}
			// Add ports based on links connected to the ATE
			for _, link := range data.Links {
				if link.Src.Device == deviceData.Id {
					portID := "port" + strconv.Itoa(portCounter)
					portIDs[link.Src.Port] = portCounter
					p := &bindpb.Port{
						Id:   portID,
						Name: link.Src.Port,
					}
					deviceData.Ports = append(deviceData.Ports, p)
					portCounter++
				} else {
					portID := "port" + strconv.Itoa(portCounter)
					portIDs[link.Dst.Port] = portCounter
					p := &bindpb.Port{
						Id:   portID,
						Name: link.Dst.Port,
					}
					deviceData.Ports = append(deviceData.Ports, p)
					portCounter++
				}
			}
		}
		// Set the Id based on the DeviceType at the end of the loop
		if device.Attributes.DeviceType == "DUT" {
			deviceData.Id = "dut"
		} else if device.Attributes.DeviceType == "ATE" {
			deviceData.Id = "ate"
		}
		// Add device to appropriate list based on device type
		if device.Attributes.DeviceType == "DUT" {
			bindingData.Duts = append(bindingData.Duts, deviceData)
		} else if device.Attributes.DeviceType == "ATE" {
			bindingData.Ates = append(bindingData.Ates, deviceData)
		}
	}

	log.Debug().Interface("Generated binding data", bindingData).Msg("Ondatra binding data")
	return bindingData
}

func convertToIntBool(data map[string]interface{}) {
	for _, device := range data["devices"].(map[string]interface{}) {
		attributes := device.(map[string]interface{})["attributes"].(map[string]interface{})
		for key, value := range attributes {
			switch v := value.(type) {
			case string:
				// Check if the string can be converted to an integer
				if intValue, err := strconv.Atoi(v); err == nil {
					attributes[key] = intValue
				} else if boolValue, err := strconv.ParseBool(strings.ToLower(v)); err == nil {
					// Check if the string can be converted to a boolean
					attributes[key] = boolValue
				}
			}
		}
	}
}

func convertOutput() {
	defer profile.LogFuncDuration(time.Now(), "convertOutput", "", "ondatra")

	// Read the JSON file
	data, err := os.ReadFile("output.json")
	if err != nil {
		log.Fatal().Msgf("Failed to read file: %v", err)
	}

	// Unmarshal the JSON data into a map[string]interface{}
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		log.Fatal().Msgf("Failed to unmarshal JSON: %v", err)
	}

	// Convert integer and boolean values from string to their respective types
	convertToIntBool(jsonData)

	// Marshal the modified JSON data back to a JSON string
	modifiedData, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatal().Msgf("Failed to marshal JSON: %v", err)
	}

	// Write the modified JSON back to a new file
	if err := os.WriteFile("output.json", modifiedData, 0644); err != nil {
		log.Fatal().Msgf("Failed to write file: %v", err)
	}

	log.Info().Msg("Data converted and written to modified_data.json successfully.")
	log.Debug().Interface("Ondatra converted output", modifiedData).Msg("")
}

func OndatraMain() (string, error) {
	defer profile.LogFuncDuration(time.Now(), "OndatraMain", "", "ondatra")

	convertOutput()
	// Load the input JSON file
	jsonData, err := os.ReadFile("output.json")
	// Set the JSON data as an environment variable
	os.Setenv("JSON_DATA", string(jsonData))
	if err != nil {
		log.Fatal().Msgf("Error reading JSON file: %v", err)
	}

	// Unmarshal the JSON data into a TestData struct
	var testData TestData
	if err := json.Unmarshal(jsonData, &testData); err != nil {
		log.Fatal().Msgf("Error unmarshalling JSON: %v", err)
	}
	// Generate the binding content
	bindingContent := generateBindingContent(testData)
	// Marshal the binding content to text format
	bindingText, err := prototext.Marshal(bindingContent)
	if err != nil {
		return "", fmt.Errorf("error writing to ondatra.binding: %w", err)
	} else {
		log.Info().Msg("ondatra.binding file created successfully.")
	}

	log.Info().Interface("Ondatra binding data", string(bindingText)).Msg("Ondatra output")
	return string(bindingText), nil
}

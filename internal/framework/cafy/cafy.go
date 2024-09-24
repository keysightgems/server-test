package cafy

import (
	"encoding/json"
	"fmt"
	"keysight/laas/controller/config"
	"keysight/laas/controller/internal/profile"
	"os"
	"strings"
	"time"
)

type OriginalJSON struct {
	Desc    string            `json:"desc"`
	Devices map[string]Device `json:"devices"`
	Links   []Link            `json:"links"`
}

type Device struct {
	Name       string     `json:"name"`
	Attributes Attributes `json:"attributes"`
	Ports      map[string]Port
	Handles    []Handle    `json:"handles"`
	Interfaces []Interface `json:"interfaces"`
	Id         string      `json:"id"`
}

type Attributes struct {
	Reserved         string `json:"reserved"`
	Role             string `json:"role"`
	Type             string `json:"type"`
	Vendor           string `json:"vendor"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Address          string `json:"address"`
	TelnetAddress    string `json:"virtual_address,omitempty"`
	TelnetMask       string `json:"virtual_mask,omitempty"`
	TelnetName       string `json:"virtual_name,omitempty"`
	TelnetPort       string `json:"virtual_port,omitempty"`
	TelnetInterface  string `json:"virtual_interface,omitempty"`
	Rp0Address       string `json:"rp0_address,omitempty"`
	Rp0Mask          string `json:"rp0_mask,omitempty"`
	Rp0Name          string `json:"rp0_name,omitempty"`
	Rp0Port          string `json:"rp0_port,omitempty"`
	Rp0Interface     string `json:"rp0_interface,omitempty"`
	ConsoleAddress   string `json:"console_address,omitempty"`
	ConsoleMask      string `json:"console_mask,omitempty"`
	ConsoleName      string `json:"console_name,omitempty"`
	ConsolePort      string `json:"console_port,omitempty"`
	ConsoleInterface string `json:"console_interface,omitempty"`
	TelnetDefault    string `json:"telnet_default,omitempty"`
	SshDefault       string `json:"ssh_default,omitempty"`
	ConsoleDefault   string `json:"console_default,omitempty"`
	YdkDefault       string `json:"ydk_default,omitempty"`
	TelnetVia        string `json:"telnet_via,omitempty"`
	SshVia           string `json:"ssh_via,omitempty"`
	ConsoleVia       string `json:"console_via,omitempty"`
	YdkVia           string `json:"ydk_via,omitempty"`
	OS               string `json:"os,omitempty"`
	Platform         string `json:"platform,omitempty"`
	DefaultName      string `json:"default_name,omitempty"`
	DefaultUsername  string `json:"default_username,omitempty"`
	DefaultPassword  string `json:"default_password,omitempty"`
	TgnServerType    string `json:"tgn_server_type,omitempty"`
	TgnServerUser    string `json:"tgn_server_user,omitempty"`
	TgnServerPw      string `json:"tgn_server_pw,omitempty"`
	ChassisIP        string `json:"chassis_ip,omitempty"`
	ServerIP         string `json:"server_ip,omitempty"`
	TelnetConnection string `json:"telnet_connection,omitempty"`
	SshConnection    string `json:"ssh_connection,omitempty"`
	HaConnection     string `json:"ha_connection,omitempty"`
	YdkConnection    string `json:"ydk_connection,omitempty"`
}

type Port struct {
	Name       string `json:"name"`
	Attributes Attributes
}

type Link struct {
	Src LinkEndpoint `json:"src"`
	Dst LinkEndpoint `json:"dst"`
}

type LinkEndpoint struct {
	Device string `json:"device"`
	Port   string `json:"port"`
}

type Handle struct {
	Connection    string      `json:"connection,omitempty"`
	Credential    string      `json:"credential,omitempty"`
	Name          string      `json:"name,omitempty"`
	Via           string      `json:"via,omitempty"`
	DefaultHandle interface{} `json:"default,omitempty"`
}

type Interface struct {
	Alias     string `json:"alias"`
	Interface string `json:"interface"`
	Link      string `json:"link"`
}

type AccessInfo struct {
	AddressInfo []AddressInfo `json:"address_info,omitempty"`
	Interface   string        `json:"interface,omitempty"`
	Name        string        `json:"name,omitempty"`
}

type AddressInfo struct {
	Address string      `json:"address,omitempty"`
	Mask    string      `json:"mask,omitempty"`
	Name    string      `json:"name,omitempty"`
	Port    interface{} `json:"port,omitempty"`
}

type NewJSON struct {
	Credentials []struct {
		Name     string `json:"name,omitempty"`
		Password string `json:"password,omitempty"`
		Username string `json:"username,omitempty"`
	} `json:"credentials"`
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
}

type Node struct {
	AccessInfo    []AccessInfo `json:"access_info,omitempty"`
	Alias         string       `json:"alias"`
	Handles       []Handle     `json:"handles,omitempty"`
	ID            int          `json:"id"`
	Interfaces    []Interface  `json:"interfaces,omitempty"`
	Name          string       `json:"name"`
	OS            string       `json:"os,omitempty"`
	Platform      string       `json:"platform,omitempty"`
	Type          string       `json:"type,omitempty"`
	TgnServerType string       `json:"tgn_server_type,omitempty"`
	TgnServerUser string       `json:"tgn_server_user,omitempty"`
	TgnServerPw   string       `json:"tgn_server_pw,omitempty"`
	ChassisIP     string       `json:"chassis_ip,omitempty"`
	ServerIP      string       `json:"server_ip,omitempty"`
}

var log = config.GetLogger("cafy")

func convertJSON(originalData OriginalJSON) NewJSON {
	defer profile.LogFuncDuration(time.Now(), "convertJSON", "", "cafy")

	// This function converts generated output.json to Cafyq_testbed.json
	log.Info().Msg("Invoked convertJSON")
	newData := NewJSON{
		Credentials: make([]struct {
			Name     string `json:"name,omitempty"`
			Password string `json:"password,omitempty"`
			Username string `json:"username,omitempty"`
		}, 0),
		Name:  "Sample-Topo",
		Nodes: make([]Node, 0),
	}

	// Helper function to convert device to credentials
	deviceToCredentials := func(device Device) *struct {
		Name     string `json:"name,omitempty"`
		Password string `json:"password,omitempty"`
		Username string `json:"username,omitempty"`
	} {
		defaultName := stringConversion(device.Attributes.DefaultName, "string").(string)
		defaultPassword := stringConversion(device.Attributes.DefaultPassword, "string").(string)
		defaultUsername := stringConversion(device.Attributes.DefaultUsername, "string").(string)
		if defaultName == "" && defaultPassword == "" && defaultUsername == "" {
			return nil
		}
		return &struct {
			Name     string `json:"name,omitempty"`
			Password string `json:"password,omitempty"`
			Username string `json:"username,omitempty"`
		}{
			Name:     defaultName,
			Password: defaultPassword,
			Username: defaultUsername,
		}
	}
	var dutCounter, ateCounter, idcounter int = 1, 1, 1
	aliasMap := make(map[string]string)
	// Helper function to convert device to node
	deviceToNode := func(device Device, links []Link) Node {
		node := Node{
			AccessInfo: make([]AccessInfo, 0),
			Handles:    make([]Handle, 0),
			ID:         idcounter,
			Interfaces: make([]Interface, 0),
			Name:       device.Id,
		}
		os := stringConversion(device.Attributes.OS, "string").(string)
		node.OS = os
		ctype := stringConversion(device.Attributes.Type, "string").(string)
		node.Type = ctype
		idcounter++
		platform := stringConversion(device.Attributes.Platform, "string").(string)
		node.Platform = platform
		if strings.ToLower(device.Attributes.Role) == "dut" {
			node.Alias = fmt.Sprintf("R%d", dutCounter)
			if node.Alias != device.Id {
				aliasMap[device.Id] = node.Alias
			} else {
				aliasMap[device.Id] = device.Id
			}
			dutCounter++
		} else {
			node.TgnServerType = device.Attributes.TgnServerType
			node.TgnServerUser = device.Attributes.TgnServerUser
			node.TgnServerPw = device.Attributes.TgnServerPw
			node.ChassisIP = device.Attributes.ChassisIP
			node.ServerIP = device.Attributes.ServerIP
			node.Alias = fmt.Sprintf("T%d", ateCounter)
			if node.Alias != device.Id {
				aliasMap[device.Id] = node.Alias
			} else {
				aliasMap[device.Id] = device.Id
			}
			ateCounter++
		}
		if strings.ToLower(device.Attributes.Role) != "ate" {
			if device.Attributes.TelnetAddress != "" && device.Attributes.TelnetAddress != "null" {
				addAccessInfoIfTelnetPresent(&node, device)
			}
			if device.Attributes.Rp0Address != "" && device.Attributes.Rp0Address != "null" {
				addAccessInfoIfRp0Present(&node, device)
			}
			if device.Attributes.ConsoleAddress != "" && device.Attributes.ConsoleAddress != "null" {
				addAccessInfoIfConsolePresent(&node, device)
			}
			if device.Attributes.TelnetConnection != "" && device.Attributes.TelnetConnection != "null" {
				telnetConnection(&node, device)
			}
			if device.Attributes.SshConnection != "" && device.Attributes.SshConnection != "null" {
				sshConnection(&node, device)
			}
			if device.Attributes.HaConnection != "" && device.Attributes.HaConnection != "null" {
				haConnection(&node, device)
			}
			if device.Attributes.YdkConnection != "" && device.Attributes.YdkConnection != "null" {
				ydkConnection(&node, device)
			}

		}
		input := fmt.Sprintf("%s", links)
		linksp := generateStructLink(input)
		ifaceNames := GenerateInterfaceMap(linksp, aliasMap[device.Id], device.Attributes.Role)
		if strings.ToLower(device.Attributes.Role) == "ate" {
			node.Alias = "TGEN"
		}
		for key, value := range ifaceNames {
			if key == aliasMap[device.Id] {
				for dkey, dvalue := range value {
					node.Interfaces = append(node.Interfaces, Interface{
						Alias:     dvalue,
						Interface: dkey,
						Link:      dvalue,
					})
				}
			}
		}
		return node
	}

	for _, device := range originalData.Devices {
		credentials := deviceToCredentials(device)
		if credentials != nil {
			newData.Credentials = append(newData.Credentials, *credentials)
		}
		newData.Nodes = append(newData.Nodes, deviceToNode(device, originalData.Links))
	}

	log.Debug().Interface("Converted data", newData).Msg("Cafy converted json")
	return newData
}

func addAccessInfoIfTelnetPresent(node *Node, device Device) {
	conPort := stringConversion(device.Attributes.TelnetPort, "int")
	telnetAddress := stringConversion(device.Attributes.TelnetAddress, "string").(string)
	telnetMask := stringConversion(device.Attributes.TelnetMask, "string").(string)
	telnetName := stringConversion(device.Attributes.TelnetName, "string").(string)
	telnetInterface := stringConversion(device.Attributes.TelnetInterface, "string").(string)
	addressInfo := AddressInfo{
		Address: telnetAddress,
		Mask:    telnetMask,
		Name:    telnetName,
	}
	if conPort != 0 {
		addressInfo.Port = conPort
	}
	node.AccessInfo = append(node.AccessInfo, AccessInfo{
		AddressInfo: []AddressInfo{addressInfo},
		Interface:   telnetInterface,
		Name:        telnetInterface,
	})
}
func addAccessInfoIfRp0Present(node *Node, device Device) {
	conPort := stringConversion(device.Attributes.Rp0Port, "int")
	rp0Address := stringConversion(device.Attributes.Rp0Address, "string").(string)
	rp0Mask := stringConversion(device.Attributes.Rp0Mask, "string").(string)
	rp0Name := stringConversion(device.Attributes.Rp0Name, "string").(string)
	rp0Interface := stringConversion(device.Attributes.Rp0Interface, "string").(string)
	addressInfo := AddressInfo{
		Address: rp0Address,
		Mask:    rp0Mask,
		Name:    rp0Name,
	}
	if conPort != 0 {
		addressInfo.Port = conPort
	}
	node.AccessInfo = append(node.AccessInfo, AccessInfo{
		AddressInfo: []AddressInfo{addressInfo},
		Interface:   rp0Interface,
		Name:        rp0Interface,
	})
}

func addAccessInfoIfConsolePresent(node *Node, device Device) {
	conPort := stringConversion(device.Attributes.ConsolePort, "int")
	consoleAddress := stringConversion(device.Attributes.ConsoleAddress, "string").(string)
	consoleMask := stringConversion(device.Attributes.ConsoleMask, "string").(string)
	consoleName := stringConversion(device.Attributes.ConsoleName, "string").(string)
	consoleInterface := stringConversion(device.Attributes.ConsoleInterface, "string").(string)
	addressInfo := AddressInfo{
		Address: consoleAddress,
		Mask:    consoleMask,
		Name:    consoleName,
	}
	if conPort != 0 {
		addressInfo.Port = conPort
	}

	node.AccessInfo = append(node.AccessInfo, AccessInfo{
		AddressInfo: []AddressInfo{addressInfo},
		Interface:   consoleInterface,
		Name:        consoleInterface,
	})
}

// This function converts generated output.json to Cafyq_testbed.json
func CafyMain() (string, error) {
	defer profile.LogFuncDuration(time.Now(), "CafyMain", "", "cafy")
	updateOutputJson()
	// Specify the path to your JSON file
	filePath := "output.json"

	// Read the JSON data from the file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Parse the JSON data into the OriginalJSON struct
	var originalData OriginalJSON
	if err := json.Unmarshal(jsonData, &originalData); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON file: %w", err)
	}

	// Convert the originalData to the desired format
	newData := convertJSON(originalData)
	resultJSON, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to convert data to JSON file: %w", err)
	}
	// Specify the path for the output JSON file
	outputFilePath := "cafy_testbed.json"

	// Write the result JSON to the output file
	err = os.WriteFile(outputFilePath, resultJSON, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write data to output file: %w", err)
	}
	log.Info().Msg("Successfully generated cafy testbed file")
	// Read the content of the generated file
	fileContent, err := os.ReadFile(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %w", err)
	}
	// Print the content of the generated file
	//log.Debug().Msgf("Generated cafy_testbed.json content: %s", string(fileContent))
	// Respond to the client with the result JSON using NewJSON
	// c.IndentedJSON(http.StatusOK, NewJSON{
	// 	Credentials: newData.Credentials,
	// 	Name:        "Sample_TOPO",
	// 	Nodes:       newData.Nodes,
	// })

	log.Info().Interface("Cafy testbed data", string(fileContent)).Msg("Cafy output")
	return string(fileContent), nil
}

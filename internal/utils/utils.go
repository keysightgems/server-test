package utils

import (
	"fmt"
	"keysight/laas/controller/internal/api"
	"strconv"
	"strings"
)

// ParseAddr parses a string of following
// formats: "1.1.1.1:8000" or "[1:1:1:1:1:1:1:1]:8000" or "host:8000"
// and returns Addr
func ParseAddr(addr string) (*api.Addr, error) {
	if len(addr) == 0 {
		return nil, fmt.Errorf("address cannot be empty")
	}

	uAddr := api.Addr{}
	lastColonIndex := strings.LastIndex(addr, ":")
	if lastColonIndex == -1 {
		return nil, fmt.Errorf("both hostname/IP and port must be provided in input address, e.g. localhost:8000 or 1.1.1.1:8000")
	}

	if uAddr.Host = addr[:lastColonIndex]; len(uAddr.Host) == 0 {
		return nil, fmt.Errorf("hostname/IP cannot be empty")
	}

	port, err := strconv.ParseUint(addr[lastColonIndex+1:], 10, 32)
	if err != nil {
		return nil, err
	}
	uAddr.Port = uint32(port)

	return &uAddr, nil
}

func SplitString(input string) (string, string) {
	// Added function to split the device, port and return
	stringSplits := strings.Split(input, ":")
	if len(stringSplits) == 2 {
		return stringSplits[0], stringSplits[1]
	}
	// If there are not exactly two parts, you can handle the error accordingly.
	return "", ""
}

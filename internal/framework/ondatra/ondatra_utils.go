package ondatra

import (
	"strconv"

	bindpb "github.com/openconfig/featureprofiles/topologies/proto/binding"
)

func checkTypeAndReturn(value string, port interface{}) string {
	switch v := port.(type) {
	case float64:
		return value + ":" + strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return value + ":" + strconv.Itoa(v)
	case string:
		if v != "null" {
			return value + ":" + v
		} else {
			return value
		}
	default:
		return value
	}
}

func getBoolValue(value interface{}) bool {
	if value == nil {
		return false
	}
	b, ok := value.(bool)
	if !ok {
		return false
	}
	return b
}

func stringTest(value string) string {
	if value == "null" {
		return ""
	}
	return value
}

func boolTest(value interface{}) interface{} {
	if value == "null" {
		return ""
	}
	return value
}

func deviceOptions(options *bindpb.Options, device Device) *bindpb.Options {
	if stringTest(device.Attributes.DutOptionsUser) != "" {
		options.Username = device.Attributes.DutOptionsUser
	}
	if stringTest(device.Attributes.DutOptionsPass) != "" {
		options.Password = device.Attributes.DutOptionsPass
	}
	if boolTest(device.Attributes.OptionsInsecure) != "" {
		options.Insecure = getBoolValue(device.Attributes.OptionsInsecure)
	}
	if boolTest(device.Attributes.DutOptionsSkipVerify) != "" {
		options.SkipVerify = getBoolValue(device.Attributes.DutOptionsSkipVerify)
	}
	return options
}

func isNull(value interface{}) bool {
	if value == nil {
		return true
	}
	if str, ok := value.(string); ok && (str == "null" || str == "") {
		return true
	}
	return false
}

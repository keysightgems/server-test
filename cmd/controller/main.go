package main

import (
	"keysight/laas/controller/config"
	// "keysight/laas/controller/internal/controller"
	// inventory "keysight/laas/controller/internal/inventory/netbox"
	"keysight/laas/controller/internal/service"

	httpsvc "keysight/laas/controller/internal/service/http"
	// graph "github.com/openconfig/ondatra/binding/portgraph"
	"keysight/laas/controller/internal/timelimited"
	"keysight/laas/controller/internal/validator"
)

var log config.Logger

func main() {
	// Parse command-line flags
	config.ParseFlags()

	log = config.GetLogger("main")
	log.Info().Msg("Parsed flags and initialized logger from main")

	log.Info().
		Interface("config", config.Config).
		Msg("Initialized application config")

	// // Get inventory
	// inventory.GetCreateInvFromNetbox(*config.Config.NetboxApiURL, *config.Config.NetboxUserToken)

	// // This line is for debugging purpose only
	// // testbedData := LoadTestbedData("testbed.json")
	// // Convert Inventory Data type from other types to string
	// controller.ConvertInventoryDataType()
	// // Create Concrete Graph
	// controller.InventoryConfig = controller.LoadInventoryData("inventory.json")
	// controller.InventoryGraph = graph.ConcreteGraph{}
	// controller.ConfigNodesToDevices = map[*graph.ConcreteNode]controller.Device{}
	// controller.ConfigPortsToPorts = map[*graph.ConcretePort]controller.Port{}
	// controller.LoadConcreteGraph()

	// For timelimited binary, spawn build validity checker go-routine
	if validator.TimeExpiryCheck {
		if err := timelimited.SpawnTimeExpiryChecker(); err != nil {
			log.Fatal().Err(err).Msg("Failed starting License checker")
		}
	}

	// Initialize log to stdout
	if err := config.InitStdoutLoggers(); err != nil {
		log.Error().Err(err).Msg("Failed starting streaming logs to stdout")
	}

	termChan := service.TerminationChannels{
		StopHttpServer: make(chan bool),
	}

	// Initialize https server
	termChan.ErrHttpServer = httpsvc.ServeHTTP(termChan.StopHttpServer)

	service.WaitForTermination(&termChan)
}

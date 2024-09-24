package service

import (
	"keysight/laas/controller/config"
	"keysight/laas/controller/internal/timelimited"
	"keysight/laas/controller/internal/validator"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var log = config.GetLogger("service")

type TerminationChannels struct {
	StopHttpServer chan bool
	ErrHttpServer  chan error
}

func WaitForTermination(termChan *TerminationChannels) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	activeGoRoutines := 1
	log.Info().Msg("Waiting for active routines")

	for {
		select {
		case sig := <-sigChan:
			log.Warn().Interface("signal", sig).Msg("Caught Signal")
			termChan.StopHttpServer <- true
		case err := <-termChan.ErrHttpServer:
			log.Error().Err(err).Msg("Error in HTTP Server")
			activeGoRoutines -= 1
		}

		if activeGoRoutines == 0 {
			log.Warn().Msg("Terminated main process")
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func GetTimeExpiryStatus() error {
	if validator.TimeExpiryCheck {
		if err := timelimited.IsTimeStampValid(); err != nil {
			return err
		}
	}
	return nil
}

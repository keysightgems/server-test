package profile

import (
	"keysight/laas/controller/config"
	"time"
)

var log = config.GetLogger("profile")

// LogFuncDuration logs the time difference between the input start time and current time
func LogFuncDuration(start time.Time, apiName string, apiChoice string, apiScope string) int64 {
	done := time.Since(start)

	log.Info().
		Str("api", apiName).
		Str("choice", apiChoice).
		Str("scope", apiScope).
		Int64("nanoseconds", done.Nanoseconds()).
		Msg("")

	return done.Nanoseconds()
}

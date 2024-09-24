package timelimited

import (
	"fmt"
	"keysight/laas/controller/config"
	"math"
	"time"
)

/* *** NOTE ***
	Set expiryTime variable and trigger a new build - changing only (YYYY, MM, DD)
 	of the time.Date format is good enough, build expires at end of set date
*/

var timelimitedConfig = struct {
	currentTime      time.Time
	expiryTime       time.Time
	sleepDuration    time.Duration
	isTimeStampValid bool
}{
	currentTime: time.Now(),
	// expiryTime:       time.Date(2024, 9, 30, 23, 59, 59, 59, time.UTC),
	expiryTime:       time.Date(2024, 10, 31, 23, 59, 59, 59, time.UTC),
	sleepDuration:    time.Duration(time.Second * 60 * 10),
	isTimeStampValid: true,
}

var log = config.GetLogger("timelimited")

func TimerExpired() bool {
	timelimitedConfig.currentTime = time.Now()
	return timelimitedConfig.currentTime.After(timelimitedConfig.expiryTime)
}

func NoOfDaysLeft(t1 time.Time, t2 time.Time) int {
	return int(math.Ceil(t2.Sub(t1).Hours()/24) - 1)
}

// SpawnTimeExpiryChecker spawns a goroutine in background that does time-expiry check
// every interval
func SpawnTimeExpiryChecker() error {
	log.Info().Msgf("Validity check initiated: %+v", timelimitedConfig.expiryTime)

	go func() {
		for {
			log.Debug().Msg("Checking for build validity")
			if TimerExpired() && timelimitedConfig.isTimeStampValid {
				timelimitedConfig.isTimeStampValid = false
				log.Error().Msg("Build is expired, please contact Keysight Support")
			}
			time.Sleep(timelimitedConfig.sleepDuration)
		}
	}()

	return nil
}

func IsTimeStampValid() error {
	days := NoOfDaysLeft(timelimitedConfig.currentTime, timelimitedConfig.expiryTime)
	if timelimitedConfig.isTimeStampValid {
		if days <= 7 {
			log.Warn().Msgf("build renewal is due in %d days,please contact Keysight Support", days)
			return nil
		}
		return nil
	} else {
		log.Error().Msg("build has expired, please contact Keysight support")
		return fmt.Errorf("build has expired, please contact Keysight support")
	}
}

// Returns if timelimited build status is expired or not
func IsBuildExpired() bool {
	return !timelimitedConfig.isTimeStampValid
}

// Sets timelimited build status as expired/not, for UT
func SetBuildExpired(val bool) {
	timelimitedConfig.isTimeStampValid = !val
}

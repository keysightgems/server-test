//go:build timelimited

package validator

// This file defines limitations that apply when the project is built with timelimited tag, in which case other files that are tagged with a different name are excluded.

// TimeExpiryCheck value defines that there is time-expiry check in timelimited build
const TimeExpiryCheck bool = true

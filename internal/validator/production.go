//go:build production

package validator

// This file defines limitations that apply when the project is built with production tag, in which case other files that are tagged with a different name are excluded.

// TimeExpiryCheck value defines that there is no time-expiry check in production build
const TimeExpiryCheck bool = false

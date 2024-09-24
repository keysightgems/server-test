//go:build development

package validator

// This file defines limitations that apply when the project is built with development tag, in which case other files that are tagged with a different name are excluded.

// TimeExpiryCheck value defines that there is no time-expiry check in development build
const TimeExpiryCheck bool = false

package constants

import "os"

// GrowthBook client keys for feature flags
const (
	// GrowthBook SDK key for internal users (production)
	GrowthBookClientKeyAnt = "sdk-xRVcrliHIlrg4og4"
	// GrowthBook SDK key for internal users (development)
	GrowthBookClientKeyAntDev = "sdk-yZQvlplybuXjYh6L"
	// GrowthBook SDK key for external users
	GrowthBookClientKeyExternal = "sdk-zAZezfDKGoZuXXKe"
)

// GetGrowthBookClientKey returns the appropriate GrowthBook client key based on user type.
func GetGrowthBookClientKey() string {
	userType := os.Getenv("USER_TYPE")
	if userType == "ant" {
		enableDev := os.Getenv("ENABLE_GROWTHBOOK_DEV")
		if enableDev == "true" || enableDev == "1" || enableDev == "yes" {
			return GrowthBookClientKeyAntDev
		}
		return GrowthBookClientKeyAnt
	}
	return GrowthBookClientKeyExternal
}

package constants

import (
	"os"
)

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
// Lazy read so ENABLE_GROWTHBOOK_DEV from globalSettings.env (applied after
// module load) is picked up. USER_TYPE is a build-time define so it's safe.
func GetGrowthBookClientKey() string {
	userType := os.Getenv("USER_TYPE")
	if userType == "ant" {
		enableDev := os.Getenv("ENABLE_GROWTHBOOK_DEV")
		if IsEnvTruthy(enableDev) {
			return GrowthBookClientKeyAntDev
		}
		return GrowthBookClientKeyAnt
	}
	return GrowthBookClientKeyExternal
}

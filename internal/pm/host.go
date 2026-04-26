package pm

import (
	"os"
	"strings"
)

// EnvHostPlatform is the environment variable Howl sets for in-app howl-pm runs
// so catalog and test-binary install paths stay Android-scoped without forking
// the CLI surface.
const EnvHostPlatform = "HOWL_PM_HOST_PLATFORM"

// EnvHostPlatformLegacy is the deprecated alias for backwards compatibility.
const EnvHostPlatformLegacy = "ZIDE_PM_HOST_PLATFORM"

const (
	HostPlatformAndroid = "android"
	HostPlatformHost    = "host"
)

// CurrentHostPlatform returns the normalized host execution class for howl-pm.
// Empty or unset HOWL_PM_HOST_PLATFORM (or deprecated ZIDE_PM_HOST_PLATFORM)
// means developer/generic host (not the Android in-app catalog mode).
func CurrentHostPlatform() string {
	v := strings.TrimSpace(os.Getenv(EnvHostPlatform))
	if v == "" {
		// Fall back to deprecated alias for compatibility.
		v = strings.TrimSpace(os.Getenv(EnvHostPlatformLegacy))
	}
	if v == "" {
		return HostPlatformHost
	}
	return strings.ToLower(v)
}

// AndroidCatalogActive reports whether Android-scoped catalog entries (such as
// android-test-binary) are visible and installable.
func AndroidCatalogActive() bool {
	return CurrentHostPlatform() == HostPlatformAndroid
}

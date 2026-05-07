package userland

import (
	"sort"

	"github.com/howl/howl-pm/internal/android"
)

// Provider describes a package source known by howl-pm.
type Provider struct {
	ID       string
	Platform string
	Scope    string
	Upstream string
	Summary  string
	Public   bool
}

var providerRegistry = []Provider{
	{
		ID:       "android-userland",
		Platform: android.PlatformAndroid,
		Scope:    "subset",
		Upstream: android.ProviderTermuxMain,
		Summary:  "Howl-maintained Android userland subset api",
		Public:   false,
	},
	{
		ID:       android.ProviderTermuxMain,
		Platform: android.PlatformAndroid,
		Scope:    "full",
		Upstream: "",
		Summary:  "Upstream Termux main repository provider",
		Public:   true,
	},
}

// Providers returns all registered providers for a platform.
func Providers(platform string) []Provider {
	var out []Provider
	for _, p := range providerRegistry {
		if p.Platform == platform {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// PublicProviders returns providers exposed to the user-facing CLI.
func PublicProviders(platform string) []Provider {
	all := Providers(platform)
	out := make([]Provider, 0, len(all))
	for _, p := range all {
		if p.Public {
			out = append(out, p)
		}
	}
	return out
}

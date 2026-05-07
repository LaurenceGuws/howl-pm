package pm

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/howl/howl-pm/internal/contract"
	"github.com/howl/howl-pm/internal/manifest"
)

const (
	PackageVisibilityPublic  = "public"
	PackageVisibilityPrivate = "private"

	InstallStrategyPrefixArchive   = "prefix-archive"
	InstallStrategyTermuxPackage   = "termux-package"
	InstallStrategyAndroidTestFile = "android-test-binary"
)

type PackageEntry struct {
	Name            string
	Version         string
	Provider        string
	ProviderRole    string
	Visibility      string
	InstallStrategy string
	SourcePackage   string
	SourceIndexRef  string
	ArtifactRef     string
	Summary         string
	Depends         string
	PreDepends      string
}

func PackageCatalog(source Source) []PackageEntry {
	entries := make([]PackageEntry, 0, len(source.Document.Artifacts))
	for _, artifact := range source.Document.Artifacts {
		if artifact.Kind != contract.ArtifactKindPackageEntry {
			continue
		}
		entries = append(entries, packageEntryFromArtifact(artifact))
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

func AvailablePackages(source Source) []string {
	seen := map[string]bool{}
	var packages []string
	for _, entry := range PackageCatalog(source) {
		if entry.Visibility != PackageVisibilityPublic || seen[entry.Name] {
			continue
		}
		seen[entry.Name] = true
		packages = append(packages, entry.Name)
	}
	if AndroidCatalogActive() {
		tb := testBinaryPackageNames(source)
		sort.Strings(tb)
		for _, name := range tb {
			if seen[name] {
				continue
			}
			seen[name] = true
			packages = append(packages, name)
		}
	}
	return packages
}

func FindPackage(source Source, name string, includePrivate bool) (PackageEntry, bool) {
	for _, entry := range PackageCatalog(source) {
		if entry.Name != name {
			continue
		}
		if entry.Visibility == PackageVisibilityPrivate && !includePrivate {
			return PackageEntry{}, false
		}
		return entry, true
	}
	return PackageEntry{}, false
}

func PrivateInstallEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(getenvFirst("HOWL_PM_ALLOW_PRIVATE", "ZIDE_PM_ALLOW_PRIVATE")))
	return v == "1" || v == "true" || v == "yes"
}

func packageEntryArtifact(source Source, ref string) (manifest.Artifact, error) {
	for _, artifact := range source.Document.Artifacts {
		if artifact.Name == ref {
			return artifact, nil
		}
	}
	return manifest.Artifact{}, fmt.Errorf("artifact %q not found in manifest", ref)
}

func packageEntryFromArtifact(artifact manifest.Artifact) PackageEntry {
	return PackageEntry{
		Name:            artifact.Name,
		Version:         artifact.Version,
		Provider:        artifact.Metadata["provider"],
		ProviderRole:    artifact.Metadata["provider_role"],
		Visibility:      artifact.Metadata["visibility"],
		InstallStrategy: artifact.Metadata["install_strategy"],
		SourcePackage:   artifact.Metadata["source_package"],
		SourceIndexRef:  artifact.Metadata["source_index_ref"],
		ArtifactRef:     artifact.Metadata["artifact_ref"],
		Summary:         artifact.Metadata["summary"],
		Depends:         artifact.Metadata["depends"],
		PreDepends:      artifact.Metadata["pre_depends"],
	}
}

func getenvFirst(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

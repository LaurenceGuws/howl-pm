// Package pm implements the user-facing Howl PM mobile package CLI surface.
package pm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/howl/howl-pm/internal/androidrepo"
	"github.com/howl/howl-pm/internal/contract"
	"github.com/howl/howl-pm/internal/manifest"
)

const (
	DefaultAndroidDevManifestURL = "https://github.com/LaurenceGuws/howl-pm/releases/download/android-dev-2026.05.07.011620/android-dev-prefix.release.manifest.json"
	DevBaselinePackage           = "dev-baseline"
)

type Source struct {
	Location string
	Document manifest.Document
}

type PrefixArtifact struct {
	Artifact manifest.Artifact
	URL      string
}

type InstallResult struct {
	Package       string
	Prefix        string
	Manifest      string
	Provider      string
	Version       string
	InstalledPath string
	FileCount     int
	DirCount      int
	SymlinkCount  int
}

type InstallStamp struct {
	InstalledAt string                `json:"installed_at"`
	Package     string                `json:"package,omitempty"`
	Manifest    string                `json:"manifest"`
	Artifact    string                `json:"artifact,omitempty"`
	Version     string                `json:"version,omitempty"`
	Provider    string                `json:"provider,omitempty"`
	Files       int                   `json:"files,omitempty"`
	Dirs        int                   `json:"dirs,omitempty"`
	Symlinks    int                   `json:"symlinks,omitempty"`
	Packages    []InstallStampPackage `json:"packages,omitempty"`
}

type InstallStampPackage struct {
	Package     string `json:"package"`
	Artifact    string `json:"artifact"`
	Version     string `json:"version"`
	Provider    string `json:"provider"`
	Files       int    `json:"files"`
	Dirs        int    `json:"dirs"`
	Symlinks    int    `json:"symlinks"`
	InstalledAt string `json:"installed_at,omitempty"`
}

func artifactCacheSuffix(artifact manifest.Artifact) string {
	switch artifact.Kind {
	case contract.ArtifactKindTestBinary:
		return ".bin"
	case contract.ArtifactKindTermuxDeb:
		return ".deb"
	case contract.ArtifactKindPackageIndex, contract.ArtifactKindPackageEntry:
		return ".json"
	default:
		return ".tar.gz"
	}
}

func AndroidPrefixArtifact(source Source) (PrefixArtifact, error) {
	var selected []manifest.Artifact
	for _, artifact := range source.Document.Artifacts {
		if artifact.Kind == contract.ArtifactKindPrefixArchive {
			selected = append(selected, artifact)
		}
	}
	if len(selected) != 1 {
		return PrefixArtifact{}, fmt.Errorf("manifest must contain exactly one %s, found %d", contract.ArtifactKindPrefixArchive, len(selected))
	}
	artifact := selected[0]
	if artifact.Metadata["archive_root"] != "usr" {
		return PrefixArtifact{}, fmt.Errorf("unsupported archive_root %q", artifact.Metadata["archive_root"])
	}
	if artifact.Metadata["provider"] == "" {
		return PrefixArtifact{}, errors.New("android-prefix-archive missing provider metadata")
	}
	artifactURL, err := ResolveURL(source.Location, artifact.URL)
	if err != nil {
		return PrefixArtifact{}, err
	}
	return PrefixArtifact{Artifact: artifact, URL: artifactURL}, nil
}

func InstallDevBaseline(ctx context.Context, source Source, prefix string, cacheDir string) (InstallResult, error) {
	if filepath.Clean(prefix) == "." || strings.TrimSpace(prefix) == "" {
		return InstallResult{}, errors.New("prefix must not be empty")
	}
	artifact, err := AndroidPrefixArtifact(source)
	if err != nil {
		return InstallResult{}, err
	}
	archivePath, err := FetchArtifact(ctx, artifact.Artifact, artifact.URL, cacheDir)
	if err != nil {
		return InstallResult{}, err
	}
	stats, err := ExtractUSRToPrefix(archivePath, prefix)
	if err != nil {
		return InstallResult{}, err
	}
	if err := writeInstallStamp(prefix, source, InstallStampPackage{
		Package:     DevBaselinePackage,
		Artifact:    artifact.Artifact.Name,
		Version:     artifact.Artifact.Version,
		Provider:    artifact.Artifact.Metadata["provider"],
		Files:       stats.files,
		Dirs:        stats.dirs,
		Symlinks:    stats.symlinks,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		Package:       DevBaselinePackage,
		Prefix:        prefix,
		Manifest:      source.Location,
		Provider:      artifact.Artifact.Metadata["provider"],
		Version:       artifact.Artifact.Version,
		InstalledPath: archivePath,
		FileCount:     stats.files,
		DirCount:      stats.dirs,
		SymlinkCount:  stats.symlinks,
	}, nil
}

func InstallPackage(ctx context.Context, source Source, packageName string, prefix string, cacheDir string) (InstallResult, error) {
	if entry, ok := FindPackage(source, packageName, PrivateInstallEnabled()); ok {
		switch entry.InstallStrategy {
		case InstallStrategyPrefixArchive:
			return InstallPackageEntryPrefixArchive(ctx, source, entry, prefix, cacheDir)
		case InstallStrategyTermuxPackage:
			return InstallTermuxPackage(ctx, source, entry, prefix, cacheDir)
		case InstallStrategyAndroidTestFile:
			return InstallAndroidTestBinary(ctx, source, entry.ArtifactRef, prefix, cacheDir)
		default:
			return InstallResult{}, fmt.Errorf("unsupported install strategy %q for %s", entry.InstallStrategy, packageName)
		}
	}
	return InstallAndroidTestBinary(ctx, source, packageName, prefix, cacheDir)
}

func InstallPackageEntryPrefixArchive(ctx context.Context, source Source, entry PackageEntry, prefix string, cacheDir string) (InstallResult, error) {
	artifact, err := packageEntryArtifact(source, entry.ArtifactRef)
	if err != nil {
		return InstallResult{}, err
	}
	artifactURL, err := ResolveURL(source.Location, artifact.URL)
	if err != nil {
		return InstallResult{}, err
	}
	archivePath, err := FetchArtifact(ctx, artifact, artifactURL, cacheDir)
	if err != nil {
		return InstallResult{}, err
	}
	stats, err := ExtractUSRToPrefix(archivePath, prefix)
	if err != nil {
		return InstallResult{}, err
	}
	stampPkg := InstallStampPackage{
		Package:     entry.Name,
		Artifact:    artifact.Name,
		Version:     artifact.Version,
		Provider:    entry.Provider,
		Files:       stats.files,
		Dirs:        stats.dirs,
		Symlinks:    stats.symlinks,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeInstallStamp(prefix, source, stampPkg); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		Package:       entry.Name,
		Prefix:        prefix,
		Manifest:      source.Location,
		Provider:      entry.Provider,
		Version:       artifact.Version,
		InstalledPath: archivePath,
		FileCount:     stats.files,
		DirCount:      stats.dirs,
		SymlinkCount:  stats.symlinks,
	}, nil
}

func InstallTermuxPackage(ctx context.Context, source Source, entry PackageEntry, prefix string, cacheDir string) (InstallResult, error) {
	indexArtifact, err := packageEntryArtifact(source, entry.SourceIndexRef)
	if err != nil {
		return InstallResult{}, err
	}
	index, err := loadIndexFromArtifact(ctx, source, indexArtifact)
	if err != nil {
		return InstallResult{}, err
	}
	rootName := entry.SourcePackage
	if rootName == "" {
		rootName = entry.Name
	}
	packages, err := androidrepo.ResolveClosure(index, []string{rootName})
	if err != nil {
		return InstallResult{}, err
	}
	baseURL := indexArtifact.Metadata["base_url"]
	if baseURL == "" {
		return InstallResult{}, fmt.Errorf("index artifact %q missing base_url metadata", indexArtifact.Name)
	}

	stagingRoot, err := os.MkdirTemp("", "howl-pm-termux-stage-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(stagingRoot)

	var totals extractStats
	var lastPath string
	for _, pkg := range packages {
		packageURL, err := androidrepo.AbsolutePackageURL(baseURL, pkg.Filename)
		if err != nil {
			return InstallResult{}, err
		}
		artifact := manifest.Artifact{
			Name:    contract.ProviderTermuxMain + "/" + pkg.Name,
			Kind:    contract.ArtifactKindTermuxDeb,
			Version: pkg.Version,
			URL:     packageURL,
			SHA256:  pkg.SHA256,
			Size:    pkg.Size,
			Metadata: map[string]string{
				"provider":              entry.Provider,
				"provider_role":         entry.ProviderRole,
				"provider_platform":     source.Document.Platform,
				"provider_architecture": indexArtifact.Metadata["provider_architecture"],
				"package":               pkg.Name,
			},
		}
		lastPath, err = FetchArtifact(ctx, artifact, packageURL, cacheDir)
		if err != nil {
			return InstallResult{}, err
		}
		stats, err := installDebIntoPrefix(lastPath, stagingRoot, prefix)
		if err != nil {
			return InstallResult{}, err
		}
		totals.files += stats.files
		totals.dirs += stats.dirs
		totals.symlinks += stats.symlinks
	}

	stampPkg := InstallStampPackage{
		Package:     entry.Name,
		Artifact:    entry.SourceIndexRef,
		Version:     entry.Version,
		Provider:    entry.Provider,
		Files:       totals.files,
		Dirs:        totals.dirs,
		Symlinks:    totals.symlinks,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeInstallStamp(prefix, source, stampPkg); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		Package:       entry.Name,
		Prefix:        prefix,
		Manifest:      source.Location,
		Provider:      entry.Provider,
		Version:       entry.Version,
		InstalledPath: lastPath,
		FileCount:     totals.files,
		DirCount:      totals.dirs,
		SymlinkCount:  totals.symlinks,
	}, nil
}

func FetchArtifact(ctx context.Context, artifact manifest.Artifact, artifactURL string, cacheDir string) (string, error) {
	if cacheDir == "" {
		cacheDir = DefaultCacheDir()
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	cacheName := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(
		artifact.Name + "-" + artifact.Version + "-" + artifact.SHA256[:12] + artifactCacheSuffix(artifact),
	)
	cachePath := filepath.Join(cacheDir, cacheName)
	if verifyPath(cachePath, artifact.Size, artifact.SHA256) {
		return cachePath, nil
	}

	tempPath := cachePath + ".tmp"
	if IsURL(artifactURL) {
		if err := downloadURL(ctx, artifactURL, tempPath); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
	} else {
		if err := copyFile(artifactURL, tempPath); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
	}
	if !verifyPath(tempPath, artifact.Size, artifact.SHA256) {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("artifact verification failed for %s", artifact.Name)
	}
	if err := os.Rename(tempPath, cachePath); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return cachePath, nil
}

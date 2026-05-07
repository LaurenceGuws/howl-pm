// Package pm implements the user-facing Howl PM mobile package CLI surface.
package pm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/howl/howl-pm/internal/androidprefix"
	"github.com/howl/howl-pm/internal/androidrepo"
	"github.com/howl/howl-pm/internal/manifest"
)

const (
	DefaultAndroidDevManifestURL = "https://github.com/LaurenceGuws/howl-pm/releases/download/android-dev-2026.05.01.014509/android-dev-prefix.release.manifest.json"
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

func LoadSource(ctx context.Context, location string) (Source, error) {
	if strings.TrimSpace(location) == "" {
		return Source{}, errors.New("manifest location must not be empty")
	}

	var payload []byte
	var err error
	if IsURL(location) {
		payload, err = readURL(ctx, location)
	} else {
		payload, err = os.ReadFile(filepath.Clean(location))
	}
	if err != nil {
		return Source{}, err
	}

	var doc manifest.Document
	if err := json.Unmarshal(payload, &doc); err != nil {
		return Source{}, err
	}
	if err := doc.Validate(); err != nil {
		return Source{}, err
	}
	return Source{Location: location, Document: doc}, nil
}

func artifactCacheSuffix(artifact manifest.Artifact) string {
	switch artifact.Kind {
	case "android-test-binary":
		return ".bin"
	case "android-termux-deb":
		return ".deb"
	case "android-termux-package-index", "howl-package-entry":
		return ".json"
	}
	return ".tar.gz"
}

func LoadInstallStamp(prefix string) (InstallStamp, error) {
	if strings.TrimSpace(prefix) == "" {
		return InstallStamp{}, errors.New("prefix must not be empty")
	}
	payload, err := os.ReadFile(filepath.Join(prefix, ".zide-pm-install.json"))
	if err != nil {
		return InstallStamp{}, err
	}
	var stamp InstallStamp
	if err := json.Unmarshal(payload, &stamp); err != nil {
		return InstallStamp{}, err
	}
	if len(stamp.Packages) == 0 && stamp.Package != "" {
		stamp.Packages = []InstallStampPackage{{
			Package:     stamp.Package,
			Artifact:    stamp.Artifact,
			Version:     stamp.Version,
			Provider:    stamp.Provider,
			Files:       stamp.Files,
			Dirs:        stamp.Dirs,
			Symlinks:    stamp.Symlinks,
			InstalledAt: stamp.InstalledAt,
		}}
	}
	if len(stamp.Packages) == 0 {
		return InstallStamp{}, errors.New("install stamp missing package")
	}
	return stamp, nil
}

func AndroidPrefixArtifact(source Source) (PrefixArtifact, error) {
	var selected []manifest.Artifact
	for _, artifact := range source.Document.Artifacts {
		if artifact.Kind == "android-prefix-archive" {
			selected = append(selected, artifact)
		}
	}
	if len(selected) != 1 {
		return PrefixArtifact{}, fmt.Errorf("manifest must contain exactly one android-prefix-archive, found %d", len(selected))
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
			Name:    "termux-main/" + pkg.Name,
			Kind:    "android-termux-deb",
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

func ExtractUSRToPrefix(archivePath string, prefix string) (extractStats, error) {
	prefix = filepath.Clean(prefix)
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return extractStats{}, err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return extractStats{}, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return extractStats{}, err
	}
	defer gzipReader.Close()

	var stats extractStats
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, err
		}
		relative, ok := strings.CutPrefix(filepath.ToSlash(filepath.Clean(header.Name)), "usr/")
		if !ok || relative == "" {
			continue
		}
		target, err := safeJoin(prefix, relative)
		if err != nil {
			return stats, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)&0o777); err != nil {
				return stats, err
			}
			stats.dirs++
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return stats, err
			}
			if err := writeRegularFile(target, reader, os.FileMode(header.Mode)&0o777); err != nil {
				return stats, err
			}
			stats.files++
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return stats, err
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return stats, err
			}
			stats.symlinks++
		default:
			continue
		}
	}
	return stats, nil
}

func ResolveURL(base string, value string) (string, error) {
	if IsURL(value) {
		return value, nil
	}
	if IsURL(base) {
		parsedBase, err := url.Parse(base)
		if err != nil {
			return "", err
		}
		parsedValue, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		return parsedBase.ResolveReference(parsedValue).String(), nil
	}
	return filepath.Join(filepath.Dir(base), value), nil
}

func IsURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func DefaultCacheDir() string {
	if value := os.Getenv("HOWL_PM_CACHE"); value != "" {
		return value
	}
	// Fall back to deprecated alias for compatibility.
	if value := os.Getenv("ZIDE_PM_CACHE"); value != "" {
		return value
	}
	if value := os.Getenv("XDG_CACHE_HOME"); value != "" {
		return filepath.Join(value, "howl-pm")
	}
	if runtime.GOOS == "android" {
		if value := os.Getenv("TMPDIR"); value != "" {
			return filepath.Join(value, "howl-pm-cache")
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cache", "howl-pm")
	}
	return filepath.Join(os.TempDir(), "howl-pm-cache")
}

type extractStats struct {
	files    int
	dirs     int
	symlinks int
}

func readURL(ctx context.Context, target string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	addAuthHeaders(request)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", target, response.Status)
	}
	return io.ReadAll(response.Body)
}

func downloadURL(ctx context.Context, target string, outputPath string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	addAuthHeaders(request)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", target, response.Status)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Body)
	return err
}

func addAuthHeaders(request *http.Request) {
	request.Header.Set("User-Agent", "howl-pm")
	if request.URL.Host != "github.com" {
		return
	}
	// Check environment variables in priority order:
	// HOWL_PM_GITHUB_TOKEN (primary), ZIDE_PM_GITHUB_TOKEN (legacy),
	// GITHUB_TOKEN, GH_TOKEN, then gh CLI token.
	for _, name := range []string{"HOWL_PM_GITHUB_TOKEN", "ZIDE_PM_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if token := os.Getenv(name); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
			return
		}
	}
	if token := ghAuthToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
}

func ghAuthToken() string {
	gh, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	output, err := exec.Command(gh, "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func verifyPath(path string, wantSize int64, wantSHA256 string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if wantSize >= 0 && info.Size() != wantSize {
		return false
	}
	got, err := sha256Path(path)
	return err == nil && got == wantSHA256
}

func sha256Path(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyFile(source string, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func writeRegularFile(path string, reader io.Reader, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func safeJoin(root string, relative string) (string, error) {
	clean := filepath.Clean(relative)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path %q", relative)
	}
	target := filepath.Join(root, clean)
	resolvedRelative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if resolvedRelative == ".." || strings.HasPrefix(resolvedRelative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive path escapes prefix: %q", relative)
	}
	return target, nil
}

func loadIndexFromArtifact(ctx context.Context, source Source, artifact manifest.Artifact) (androidrepo.Index, error) {
	artifactURL, err := ResolveURL(source.Location, artifact.URL)
	if err != nil {
		return androidrepo.Index{}, err
	}
	var payload []byte
	if IsURL(artifactURL) {
		payload, err = readURL(ctx, artifactURL)
	} else {
		payload, err = os.ReadFile(filepath.Clean(artifactURL))
	}
	if err != nil {
		return androidrepo.Index{}, err
	}
	if artifact.Size >= 0 && int64(len(payload)) != artifact.Size {
		return androidrepo.Index{}, fmt.Errorf("index artifact size mismatch for %s", artifact.Name)
	}
	if got := androidrepo.HashBytes(payload); got != artifact.SHA256 {
		return androidrepo.Index{}, fmt.Errorf("index artifact sha256 mismatch for %s", artifact.Name)
	}
	return androidrepo.ParseIndex(strings.NewReader(string(payload)))
}

func installDebIntoPrefix(debPath string, stagingRoot string, prefix string) (extractStats, error) {
	if err := os.RemoveAll(stagingRoot); err != nil {
		return extractStats{}, err
	}
	if err := os.MkdirAll(stagingRoot, 0o755); err != nil {
		return extractStats{}, err
	}
	extracted, err := androidprefix.ExtractDebUSR(debPath, stagingRoot)
	if err != nil {
		return extractStats{}, err
	}
	merged, err := mergeTree(filepath.Join(stagingRoot, "usr"), prefix)
	if err != nil {
		return extractStats{}, err
	}
	_ = extracted
	return merged, nil
}

func mergeTree(sourceRoot string, targetRoot string) (extractStats, error) {
	var stats extractStats
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return stats, err
	}
	err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target, err := safeJoin(targetRoot, relative)
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(target, mode.Perm()); err != nil {
				return err
			}
			stats.dirs++
		case mode&os.ModeSymlink != 0:
			linkname, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(linkname, target); err != nil {
				return err
			}
			stats.symlinks++
		case mode.IsRegular():
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			err = writeRegularFile(target, src, mode.Perm())
			_ = src.Close()
			if err != nil {
				return err
			}
			stats.files++
		}
		return nil
	})
	return stats, err
}

func writeInstallStamp(prefix string, source Source, pkg InstallStampPackage) error {
	stamp := InstallStamp{
		InstalledAt: pkg.InstalledAt,
		Package:     pkg.Package,
		Manifest:    source.Location,
		Artifact:    pkg.Artifact,
		Version:     pkg.Version,
		Provider:    pkg.Provider,
		Files:       pkg.Files,
		Dirs:        pkg.Dirs,
		Symlinks:    pkg.Symlinks,
	}
	existing, err := LoadInstallStamp(prefix)
	if err == nil {
		stamp.Packages = append(stamp.Packages, existing.Packages...)
	}
	replaced := false
	for i := range stamp.Packages {
		if stamp.Packages[i].Package == pkg.Package {
			stamp.Packages[i] = pkg
			replaced = true
			break
		}
	}
	if !replaced {
		stamp.Packages = append(stamp.Packages, pkg)
	}
	payload, err := json.MarshalIndent(stamp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(prefix, ".zide-pm-install.json"), append(payload, '\n'), 0o644)
}

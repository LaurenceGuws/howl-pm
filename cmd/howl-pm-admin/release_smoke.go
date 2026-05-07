package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/howl/howl-pm/internal/android"
	"github.com/howl/howl-pm/internal/manifest"
)

const (
	catalogSmokeSourceRel  = "assets/howl-android-catalog-smoke.sh"
	catalogSmokeDistName   = "howl-android-catalog-smoke.sh"
	catalogSmokeArtifactID = "howl-android-catalog-smoke"
	pinnedIndexDistName    = android.IndexArtifactName
)

func moduleRootDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", cwd)
		}
		dir = parent
	}
}

func catalogSmokeReleaseAssetPath() string {
	return filepath.Join("dist", catalogSmokeDistName)
}

func materializeCatalogSmoke(distDir string) (sha256hex string, size int64, distPath string, err error) {
	root, err := moduleRootDir()
	if err != nil {
		return "", 0, "", err
	}
	srcPath := filepath.Join(root, catalogSmokeSourceRel)
	payload, err := os.ReadFile(srcPath)
	if err != nil {
		return "", 0, "", fmt.Errorf("catalog smoke source: %w", err)
	}
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return "", 0, "", err
	}
	distPath = filepath.Join(distDir, catalogSmokeDistName)
	if err := os.WriteFile(distPath, payload, 0o644); err != nil {
		return "", 0, "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), int64(len(payload)), distPath, nil
}

func applyAndroidDevReleaseEdits(doc manifest.Document, archiveAssetBaseName, indexAssetBaseName, smokeHash string, smokeSize int64) (manifest.Document, error) {
	for i := range doc.Artifacts {
		if doc.Artifacts[i].Kind == android.ArtifactKindPrefixArchive {
			doc.Artifacts[i].URL = archiveAssetBaseName
		}
		if doc.Artifacts[i].Kind == android.ArtifactKindPackageIndex {
			doc.Artifacts[i].URL = indexAssetBaseName
		}
	}
	version := "sha256-" + smokeHash[:12]
	doc.Artifacts = append(doc.Artifacts, manifest.Artifact{
		Name:    catalogSmokeArtifactID,
		Kind:    android.ArtifactKindTestBinary,
		Version: version,
		URL:     catalogSmokeDistName,
		SHA256:  smokeHash,
		Size:    smokeSize,
		Metadata: func() map[string]string {
			metadata := android.ProviderMetadata(android.ProviderRoleDevBootstrap)
			metadata["install_relative_path"] = "libexec/howl-pm/howl-android-catalog-smoke.sh"
			metadata["unix_mode"] = "0755"
			return metadata
		}(),
		Limitations: []string{
			"Development snapshot payload for howl-pm android-test-binary pull/install validation only.",
		},
	})
	doc.Notes = append(doc.Notes,
		"Artifact URLs in this release manifest are relative to the manifest location.",
		"Includes android-test-binary "+catalogSmokeArtifactID+" for Android catalog mode (HOWL_PM_HOST_PLATFORM=android).",
	)
	if err := doc.Validate(); err != nil {
		return manifest.Document{}, err
	}
	return doc, nil
}

func materializePinnedIndex(distDir, prefixManifestPath, indexPath string) (string, string, error) {
	doc, err := manifest.Load(prefixManifestPath)
	if err != nil {
		return "", "", err
	}
	if err := doc.Validate(); err != nil {
		return "", "", err
	}
	var indexArtifact *manifest.Artifact
	for i := range doc.Artifacts {
		if doc.Artifacts[i].Kind == android.ArtifactKindPackageIndex {
			indexArtifact = &doc.Artifacts[i]
			break
		}
	}
	if indexArtifact == nil {
		return "", "", fmt.Errorf("release manifest source missing %s artifact", android.ArtifactKindPackageIndex)
	}
	indexBytes, err := os.ReadFile(filepath.Clean(indexPath))
	if err != nil {
		return "", "", err
	}
	if int64(len(indexBytes)) != indexArtifact.Size {
		return "", "", fmt.Errorf("index artifact size mismatch for %s", indexArtifact.Name)
	}
	sum := sha256.Sum256(indexBytes)
	if got := hex.EncodeToString(sum[:]); got != indexArtifact.SHA256 {
		return "", "", fmt.Errorf("index artifact sha256 mismatch for %s", indexArtifact.Name)
	}
	distPath := filepath.Join(distDir, pinnedIndexDistName)
	if err := os.WriteFile(distPath, indexBytes, 0o644); err != nil {
		return "", "", err
	}
	return pinnedIndexDistName, distPath, nil
}

func writeAndroidDevReleaseManifest(prefixManifestPath, releaseManifestPath, archiveAssetBaseName, indexAssetBaseName string) error {
	hash, size, _, err := materializeCatalogSmoke(filepath.Dir(releaseManifestPath))
	if err != nil {
		return err
	}
	doc, err := manifest.Load(prefixManifestPath)
	if err != nil {
		return err
	}
	if err := doc.Validate(); err != nil {
		return err
	}
	doc, err = applyAndroidDevReleaseEdits(doc, archiveAssetBaseName, indexAssetBaseName, hash, size)
	if err != nil {
		return err
	}
	return writeManifest(releaseManifestPath, doc)
}

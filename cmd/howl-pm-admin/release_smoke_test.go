package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/howl/howl-pm/internal/android"
	"github.com/howl/howl-pm/internal/manifest"
)

func TestMaterializeCatalogSmokeWritesDist(t *testing.T) {
	dist := filepath.Join(t.TempDir(), "dist")
	hash, size, outPath, err := materializeCatalogSmoke(dist)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(dist, catalogSmokeDistName)
	if outPath != wantPath {
		t.Fatalf("outPath=%s want %s", outPath, wantPath)
	}
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(b)) != size {
		t.Fatalf("size %d vs len %d", size, len(b))
	}
	root, err := moduleRootDir()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, catalogSmokeSourceRel))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(src) {
		t.Fatal("dist payload mismatch")
	}
	if len(hash) != 64 {
		t.Fatalf("hash len %d", len(hash))
	}
}

func TestApplyAndroidDevReleaseEditsValidates(t *testing.T) {
	doc := manifest.Document{
		SchemaVersion: manifest.SchemaVersion,
		Project:       android.ProjectName,
		Platform:      android.PlatformAndroid,
		Channel:       "dev",
		Artifacts: []manifest.Artifact{
			{
				Name:    "howl-android-dev-prefix",
				Kind:    android.ArtifactKindPrefixArchive,
				Version: "sha256-aaaaaaaaaaaa",
				URL:     "howl-android-dev-prefix.tar.gz",
				SHA256:  strings.Repeat("a", 64),
				Size:    1,
				Metadata: func() map[string]string {
					metadata := android.AndroidPrefixMetadata(android.ProviderRoleDevBootstrap)
					metadata["archive_root"] = "usr"
					return metadata
				}(),
			},
			{
				Name:    android.IndexArtifactName,
				Kind:    android.ArtifactKindPackageIndex,
				Version: "sha256-bbbbbbbbbbbb",
				URL:     "Packages",
				SHA256:  strings.Repeat("b", 64),
				Size:    2,
				Metadata: func() map[string]string {
					metadata := android.ProviderMetadata(android.ProviderRoleDevBootstrap)
					metadata["base_url"] = "https://packages.termux.dev/apt/termux-main/"
					return metadata
				}(),
			},
		},
	}
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	smokeHash := strings.Repeat("c", 64)
	merged, err := applyAndroidDevReleaseEdits(doc, "howl-android-dev-prefix.tar.gz", android.IndexArtifactName, smokeHash, 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Artifacts) != 3 {
		t.Fatalf("artifacts %d", len(merged.Artifacts))
	}
	if merged.Artifacts[2].Kind != android.ArtifactKindTestBinary || merged.Artifacts[2].Name != catalogSmokeArtifactID {
		t.Fatalf("smoke artifact: %#v", merged.Artifacts[2])
	}
	if merged.Artifacts[2].SHA256 != smokeHash {
		t.Fatal("smoke hash")
	}
	if merged.Artifacts[0].URL != "howl-android-dev-prefix.tar.gz" {
		t.Fatal("archive url rewrite")
	}
	if merged.Artifacts[1].URL != android.IndexArtifactName {
		t.Fatal("index url rewrite")
	}
}

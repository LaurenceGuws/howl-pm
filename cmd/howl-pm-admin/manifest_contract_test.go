package main

import (
	"strings"
	"testing"

	"github.com/howl/howl-pm/internal/androidprefix"
	"github.com/howl/howl-pm/internal/manifest"
)

func TestNewAndroidPrefixManifestRuntimeSupportMetadataMatchesAuthority(t *testing.T) {
	doc := newAndroidPrefixManifest(
		manifest.Document{},
		"dev",
		"dist/howl-android-dev-prefix.tar.gz",
		androidprefix.ArchiveStats{
			SHA256:   strings.Repeat("a", 64),
			Size:     1,
			Files:    1,
			Dirs:     0,
			Symlinks: 0,
		},
		strings.Repeat("b", 64),
		prefixAudit{},
	)
	md := doc.Artifacts[1].Metadata
	if got, want := md["runtime_support_links"], androidprefix.PrefixArchiveRuntimeSupportLinks(); got != want {
		t.Fatalf("runtime_support_links mismatch\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := md["runtime_support_files"], androidprefix.PrefixArchiveRuntimeSupportFiles(); got != want {
		t.Fatalf("runtime_support_files mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

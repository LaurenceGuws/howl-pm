package main

import (
	"strings"
	"testing"

	"github.com/howl/howl-pm/internal/manifest"
	"github.com/howl/howl-pm/internal/prefix"
)

func TestNewAndroidPrefixManifestRuntimeSupportMetadataMatchesAuthority(t *testing.T) {
	doc := newAndroidPrefixManifest(
		manifest.Document{},
		"dev",
		"dist/howl-android-dev-prefix.tar.gz",
		prefix.ArchiveStats{
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
	if got, want := md["runtime_support_links"], prefix.PrefixArchiveRuntimeSupportLinks(); got != want {
		t.Fatalf("runtime_support_links mismatch\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := md["runtime_support_files"], prefix.PrefixArchiveRuntimeSupportFiles(); got != want {
		t.Fatalf("runtime_support_files mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

package androidprefix

import (
	"strings"
	"testing"
)

const prefixRuntimeSupportLinksGolden = "/data/data/howl.term/.z=>/data/data/howl.term/files/usr,/data/data/howl.term/ul=>/data/data/howl.term/files/usr/lib,/data/data/howl.term/ub=>/data/data/howl.term/files/usr/bin,/data/data/howl.term/b=>/data/data/howl.term/files/usr/bin,/data/data/howl.term/u/bsh=>/data/data/howl.term/files/usr/bin/sh"

const prefixRuntimeSupportFilesGolden = ""

func TestPrefixArchiveRuntimeSupportLinksGolden(t *testing.T) {
	got := PrefixArchiveRuntimeSupportLinks()
	if got != prefixRuntimeSupportLinksGolden {
		t.Fatalf("runtime_support_links drift\ngot:  %s\nwant: %s", got, prefixRuntimeSupportLinksGolden)
	}
}

func TestPrefixArchiveRuntimeSupportFilesGolden(t *testing.T) {
	got := PrefixArchiveRuntimeSupportFiles()
	if got != prefixRuntimeSupportFilesGolden {
		t.Fatalf("runtime_support_files drift\ngot:  %s\nwant: %s", got, prefixRuntimeSupportFilesGolden)
	}
}

func TestPrefixArchiveRuntimeSupportLinksEmbedsBridgeFirst(t *testing.T) {
	got := PrefixArchiveRuntimeSupportLinks()
	prefix := BinaryUSRBridgePath + "=>" + AppUSRPath
	if got != prefix && !strings.HasPrefix(got, prefix+",") {
		t.Fatalf("expected first pair %q, got %q", prefix, got)
	}
}

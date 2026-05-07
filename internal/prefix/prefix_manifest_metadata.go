package prefix

import "strings"

// PrefixArchiveRuntimeSupportFiles returns the comma-separated
// metadata.runtime_support_files value for android-prefix-archive manifests.
// The current Android prefix artifacts no longer require extra app-owned file
// aliases for etc/ or htop support paths.
func PrefixArchiveRuntimeSupportFiles() string {
	return ""
}

// PrefixArchiveRuntimeSupportLinks returns the metadata.runtime_support_links
// CSV for android-prefix-archive manifests. The first pair is always the
// BinaryUSRBridgePath symlink source mapped to AppUSRPath so it stays aligned
// with rewriteBinaryUSRRootToBridge in deb.go.
func PrefixArchiveRuntimeSupportLinks() string {
	data := "/data/data/" + AppPackageName
	parts := []string{
		BinaryUSRBridgePath + "=>" + AppUSRPath,
		data + "/ul=>" + data + "/files/usr/lib",
		data + "/ub=>" + data + "/files/usr/bin",
		data + "/b=>" + data + "/files/usr/bin",
		data + "/u/bsh=>" + data + "/files/usr/bin/sh",
	}
	return strings.Join(parts, ",")
}

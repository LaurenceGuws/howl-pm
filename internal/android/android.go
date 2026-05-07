package android

const (
	ProjectName = "howl-pm"

	PlatformAndroid = "android"
	PlatformIOS     = "ios"

	ArchitectureAArch64 = "aarch64"

	AndroidPackageName = "howl.term"
	AndroidPrefixPath  = "/data/data/" + AndroidPackageName + "/files/usr"
	AndroidTargetSDK   = "28"

	ProviderTermuxMain = "termux-main"

	ProviderRoleDevBootstrap     = "android-dev-bootstrap"
	ProviderRolePublicCatalog    = "public-catalog"
	ProviderRoleBootstrapProfile = "bootstrap-profile"

	ArtifactKindPrefixArchive = "android-prefix-archive"
	ArtifactKindPackageIndex  = "android-termux-package-index"
	ArtifactKindTermuxDeb     = "android-termux-deb"
	ArtifactKindTestBinary    = "android-test-binary"
	ArtifactKindPackageEntry  = "howl-package-entry"
	ArtifactKindIOSBundle     = "ios-bundle-manifest"

	IndexArtifactName = "termux-main-aarch64-packages-index"
)

func ProviderMetadata(role string) map[string]string {
	return map[string]string{
		"provider":              ProviderTermuxMain,
		"provider_role":         role,
		"provider_platform":     PlatformAndroid,
		"provider_architecture": ArchitectureAArch64,
	}
}

func AndroidPrefixMetadata(role string) map[string]string {
	metadata := ProviderMetadata(role)
	metadata["package_name"] = AndroidPackageName
	metadata["prefix"] = AndroidPrefixPath
	metadata["target_sdk"] = AndroidTargetSDK
	return metadata
}

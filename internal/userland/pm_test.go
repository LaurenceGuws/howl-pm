package userland

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/howl/howl-pm/internal/manifest"
)

func TestAvailablePackagesHidesTestBinariesWithoutAndroidHost(t *testing.T) {
	t.Setenv(EnvHostPlatform, "")
	doc := manifest.Document{
		SchemaVersion: manifest.SchemaVersion,
		Project:       "howl-pm",
		Platform:      "android",
		Channel:       "dev",
		Artifacts: []manifest.Artifact{
			prefixArchiveStub(),
			packageEntryStub("bash", PackageVisibilityPrivate, InstallStrategyTermuxPackage),
			packageEntryStub("htop", PackageVisibilityPublic, InstallStrategyTermuxPackage),
			testBinaryStub("tb-one", "bin/one"),
		},
	}
	source := Source{Location: "file:///tmp/x.json", Document: doc}
	got := AvailablePackages(source)
	want := []string{"htop"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestAvailablePackagesListsTestBinariesOnAndroidHost(t *testing.T) {
	t.Setenv(EnvHostPlatform, HostPlatformAndroid)
	doc := manifest.Document{
		SchemaVersion: manifest.SchemaVersion,
		Project:       "howl-pm",
		Platform:      "android",
		Channel:       "dev",
		Artifacts: []manifest.Artifact{
			prefixArchiveStub(),
			packageEntryStub("bash", PackageVisibilityPublic, InstallStrategyTermuxPackage),
			packageEntryStub("htop", PackageVisibilityPublic, InstallStrategyTermuxPackage),
			testBinaryStub("tb-b", "bin/b"),
			testBinaryStub("tb-a", "bin/a"),
		},
	}
	source := Source{Location: "file:///tmp/x.json", Document: doc}
	got := AvailablePackages(source)
	want := []string{"bash", "htop", "tb-a", "tb-b"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestInstallAndroidTestBinaryWritesPayload(t *testing.T) {
	t.Setenv(EnvHostPlatform, HostPlatformAndroid)
	work := t.TempDir()
	payloadPath := filepath.Join(work, "payload.dat")
	payload := []byte("howl-pm-test-binary-payload\n")
	if err := os.WriteFile(payloadPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	hexHash := hex.EncodeToString(sum[:])

	doc := manifest.Document{
		SchemaVersion: manifest.SchemaVersion,
		Project:       "howl-pm",
		Platform:      "android",
		Channel:       "dev",
		Artifacts: []manifest.Artifact{
			testBinaryArtifact("howl-test-payload", "payload.dat", int64(len(payload)), hexHash, "bin/smoke-test"),
		},
	}
	manifestPath := filepath.Join(work, "manifest.json")
	if err := writeManifest(manifestPath, doc); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	source, err := LoadSource(ctx, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	prefix := filepath.Join(work, "usr")
	cacheDir := filepath.Join(work, "cache")
	res, err := InstallAndroidTestBinary(ctx, source, "howl-test-payload", prefix, cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.FileCount != 1 {
		t.Fatalf("file count: %d", res.FileCount)
	}
	outPath := filepath.Join(prefix, "bin", "smoke-test")
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch")
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bits, got %v", info.Mode())
	}
}

func TestInstallAndroidTestBinaryRejectsWithoutHost(t *testing.T) {
	t.Setenv(EnvHostPlatform, "")
	work := t.TempDir()
	doc := manifest.Document{
		SchemaVersion: manifest.SchemaVersion,
		Project:       "howl-pm",
		Platform:      "android",
		Channel:       "dev",
		Artifacts: []manifest.Artifact{
			testBinaryArtifact("howl-test-payload", "payload.dat", 1, "abc", "bin/x"),
		},
	}
	manifestPath := filepath.Join(work, "manifest.json")
	if err := writeManifest(manifestPath, doc); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	source, err := LoadSource(ctx, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = InstallAndroidTestBinary(ctx, source, "howl-test-payload", filepath.Join(work, "usr"), filepath.Join(work, "c"))
	if err == nil {
		t.Fatal("expected error without android host platform")
	}
}

func TestInstallPackageRejectsPrivateWithoutOptIn(t *testing.T) {
	t.Setenv(EnvHostPlatform, HostPlatformAndroid)
	t.Setenv("HOWL_PM_ALLOW_PRIVATE", "")
	source := Source{
		Location: "file:///tmp/x.json",
		Document: manifest.Document{
			SchemaVersion: manifest.SchemaVersion,
			Project:       "howl-pm",
			Platform:      "android",
			Channel:       "dev",
			Artifacts: []manifest.Artifact{
				prefixArchiveStub(),
				packageEntryStub(DevBaselinePackage, PackageVisibilityPrivate, InstallStrategyPrefixArchive),
			},
		},
	}
	if _, err := InstallPackage(context.Background(), source, DevBaselinePackage, t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("expected private install rejection")
	}
}

func TestInstallTermuxPackageFromCatalog(t *testing.T) {
	t.Setenv(EnvHostPlatform, "")
	work := t.TempDir()
	repoDir := filepath.Join(work, "repo")
	if err := os.MkdirAll(filepath.Join(repoDir, "pool", "main", "h", "hello"), 0o755); err != nil {
		t.Fatal(err)
	}
	debPath := filepath.Join(repoDir, "pool", "main", "h", "hello", "hello_1.0_aarch64.deb")
	if err := writeDebPackage(debPath, map[string]string{
		"bin/hello": "#!/bin/sh\necho hello\n",
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(debPath)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := sha256Path(debPath)
	if err != nil {
		t.Fatal(err)
	}

	indexPath := filepath.Join(work, "Packages")
	indexBody := "Package: hello\nArchitecture: aarch64\nVersion: 1.0\nFilename: pool/main/h/hello/hello_1.0_aarch64.deb\nSize: " +
		intToString(info.Size()) + "\nSHA256: " + sum + "\nDescription: Hello package\n\n"
	if err := os.WriteFile(indexPath, []byte(indexBody), 0o644); err != nil {
		t.Fatal(err)
	}
	indexHash := sha256.Sum256([]byte(indexBody))

	doc := manifest.Document{
		SchemaVersion: manifest.SchemaVersion,
		Project:       "howl-pm",
		Platform:      "android",
		Channel:       "dev",
		Artifacts: []manifest.Artifact{
			{
				Name:    "termux-main-aarch64-packages-index",
				Kind:    "android-termux-package-index",
				Version: "1",
				URL:     "Packages",
				SHA256:  hex.EncodeToString(indexHash[:]),
				Size:    int64(len(indexBody)),
				Metadata: map[string]string{
					"provider":              "termux-main",
					"provider_role":         "android-dev-bootstrap",
					"provider_platform":     "android",
					"provider_architecture": "aarch64",
					"base_url":              repoDir + string(os.PathSeparator),
				},
			},
			packageEntryStub("hello", PackageVisibilityPublic, InstallStrategyTermuxPackage),
		},
	}
	doc.Artifacts[1].Version = "1.0"
	doc.Artifacts[1].Metadata["source_package"] = "hello"
	doc.Artifacts[1].Metadata["source_index_ref"] = "termux-main-aarch64-packages-index"
	source := Source{Location: filepath.Join(work, "manifest.json"), Document: doc}

	prefix := filepath.Join(work, "usr")
	cacheDir := filepath.Join(work, "cache")
	result, err := InstallPackage(context.Background(), source, "hello", prefix, cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Package != "hello" {
		t.Fatalf("package=%s", result.Package)
	}
	payload, err := os.ReadFile(filepath.Join(prefix, "bin", "hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(payload, []byte("echo hello")) {
		t.Fatalf("unexpected payload: %q", payload)
	}
	stamp, err := LoadInstallStamp(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(stamp.Packages) != 1 || stamp.Packages[0].Package != "hello" {
		t.Fatalf("stamp packages=%#v", stamp.Packages)
	}
}

func prefixArchiveStub() manifest.Artifact {
	return manifest.Artifact{
		Name:    "pfx",
		Kind:    "android-prefix-archive",
		Version: "1",
		URL:     "x.tar.gz",
		SHA256:  "a" + repeatChar('b', 63),
		Size:    1,
		Metadata: map[string]string{
			"archive_root":          "usr",
			"provider":              "termux-main",
			"provider_role":         "android-dev-bootstrap",
			"provider_platform":     "android",
			"provider_architecture": "aarch64",
		},
	}
}

func testBinaryStub(name, rel string) manifest.Artifact {
	return testBinaryArtifact(name, "http://example.invalid/x", 1, repeatChar('a', 64), rel)
}

func termuxDebStub(name string) manifest.Artifact {
	return manifest.Artifact{
		Name:    "termux-" + name,
		Kind:    "android-termux-deb",
		Version: "1",
		URL:     "http://example.invalid/" + name + ".deb",
		SHA256:  repeatChar('c', 64),
		Size:    1,
		Metadata: map[string]string{
			"provider":              "termux-main",
			"provider_role":         "android-dev-bootstrap",
			"provider_platform":     "android",
			"provider_architecture": "aarch64",
			"package":               name,
		},
	}
}

func packageEntryStub(name, visibility, strategy string) manifest.Artifact {
	return manifest.Artifact{
		Name:    name,
		Kind:    "howl-package-entry",
		Version: "1",
		URL:     "pkg://" + name,
		SHA256:  repeatChar('d', 64),
		Size:    int64(len(name)),
		Metadata: map[string]string{
			"provider":              "android-userland",
			"provider_role":         "public-catalog",
			"provider_platform":     "android",
			"provider_architecture": "aarch64",
			"visibility":            visibility,
			"install_strategy":      strategy,
			"source_package":        name,
			"source_index_ref":      "termux-main-aarch64-packages-index",
			"artifact_ref":          "pfx",
		},
	}
}

func testBinaryArtifact(name, url string, size int64, sha256, rel string) manifest.Artifact {
	return manifest.Artifact{
		Name:    name,
		Kind:    "android-test-binary",
		Version: "1",
		URL:     url,
		SHA256:  sha256,
		Size:    size,
		Metadata: map[string]string{
			"provider":              "termux-main",
			"provider_role":         "android-dev-bootstrap",
			"provider_platform":     "android",
			"provider_architecture": "aarch64",
			"install_relative_path": rel,
		},
	}
}

func repeatChar(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func writeManifest(path string, doc manifest.Document) error {
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func writeDebPackage(path string, files map[string]string) error {
	var payload bytes.Buffer
	gz := gzip.NewWriter(&payload)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		fullName := "data/data/com.termux/files/usr/" + name
		body := []byte(content)
		if err := tw.WriteHeader(&tar.Header{
			Name: fullName,
			Mode: 0o755,
			Size: int64(len(body)),
		}); err != nil {
			return err
		}
		if _, err := tw.Write(body); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	var ar bytes.Buffer
	ar.WriteString("!<arch>\n")
	writeArMember(&ar, "debian-binary", []byte("2.0\n"))
	writeArMember(&ar, "data.tar.gz", payload.Bytes())
	return os.WriteFile(path, ar.Bytes(), 0o644)
}

func writeArMember(buf *bytes.Buffer, name string, body []byte) {
	if len(name) > 15 {
		name = name[:15]
	}
	header := []byte(
		padRight(name+"/", 16) +
			padRight("0", 12) +
			padRight("0", 6) +
			padRight("0", 6) +
			padRight("100644", 8) +
			padRight(intToString(int64(len(body))), 10) +
			"`\n",
	)
	buf.Write(header)
	buf.Write(body)
	if len(body)%2 != 0 {
		buf.WriteByte('\n')
	}
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value[:width]
	}
	return value + string(bytes.Repeat([]byte(" "), width-len(value)))
}

func intToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

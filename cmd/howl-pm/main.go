// Command howl-pm is the user-facing Howl PM package CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/howl/howl-pm/internal/pm"
)

const version = "0.1.1-beta.1"

func main() {
	args := normalizedArgs(os.Args)
	if len(args) < 2 {
		printHelp()
		return
	}
	if args[1] == "pkg" {
		if err := pkgCommand(args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	var err error
	switch args[1] {
	case "help", "-h", "--help":
		printHelp()
	case "version":
		fmt.Println(version)
	case "doctor":
		err = doctor(args[2:])
	case "list-providers", "providers":
		err = listProviders(args[2:])
	case "list-available", "list":
		err = listAvailable(args[2:])
	case "install":
		err = install(args[2:])
	default:
		err = fmt.Errorf("unknown command %q", args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`howl-pm manages Howl PM packages from pinned artifact manifests.

Usage:
  howl-pm <command> [options]

Commands:
  pkg             Termux-style package UX (update/install/upgrade/search/show/remove).
  doctor          Validate the configured package manifest and print provider info.
  list-providers  List public provider ids available to the CLI surface.
  list-available  List packages/groups available from the manifest.
  install         Install a supported package/group into a prefix.
  version         Print the tool version.
  help            Show this help.

Android catalog (HOWL_PM_HOST_PLATFORM=android):
  android-test-binary artifacts from the manifest are listed and installable as
  additional package names (host-side runs keep this catalog off by default).

Examples:
  howl-pm pkg update
  howl-pm pkg install bash
  howl-pm doctor
  howl-pm list-providers
  howl-pm list-available
  howl-pm install bash --prefix /data/data/uk.laurencegouws.zide/files/usr
  howl-pm install bash --manifest ./android-dev-prefix.release.manifest.json --prefix ./tmp/usr

howl-pm is the product CLI surface. Provider/package internals stay behind the
manifest api.`)
}

func normalizedArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	base := filepath.Base(args[0])
	if base == "pkg" {
		out := make([]string, 0, len(args)+1)
		out = append(out, args[0], "pkg")
		out = append(out, args[1:]...)
		return out
	}
	return args
}

func pkgCommand(args []string) error {
	if len(args) == 0 {
		printPkgHelp()
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "help", "-h", "--help":
		printPkgHelp()
		return nil
	case "update", "up":
		return doctor(rest)
	case "upgrade":
		return install(append([]string{pm.DevBaselinePackage}, rest...))
	case "install", "in":
		return install(rest)
	case "remove", "rm", "uninstall":
		return fmt.Errorf("pkg remove is not implemented yet")
	case "search", "s":
		return pkgSearch(rest)
	case "show":
		return pkgShow(rest)
	case "list-all":
		return listAvailable(rest)
	case "list-installed":
		return pkgListInstalled(rest)
	default:
		return fmt.Errorf("unknown pkg command %q", sub)
	}
}

func printPkgHelp() {
	fmt.Println(`pkg wraps howl-pm with a Termux-style command shape.

Usage:
  pkg <command> [options]

Commands:
  update          Validate/refresh manifest source.
  upgrade         Reinstall internal baseline into prefix.
  install         Install a package/group.
  remove          Remove a package (not implemented yet).
  search          Search manifest package names.
  show            Show package metadata from manifest.
  list-all        List all available package names.
  list-installed  Show installed package from install stamp.`)
}

func listProviders(args []string) error {
	fs := commonFlagSet("list-providers")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for _, provider := range pm.PublicProviders("android") {
		fmt.Printf("%s\t%s\n", provider.ID, provider.Summary)
	}
	return nil
}

func doctor(args []string) error {
	fs := commonFlagSet("doctor")
	manifestPath := fs.String("manifest", pm.DefaultAndroidDevManifestURL, "artifact manifest URL/path")
	prefix := fs.String("prefix", defaultPrefix(), "installed prefix used for offline doctor fallback")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Printf("howl_pm_host_platform=%s\n", pm.CurrentHostPlatform())
	source, err := loadSource(*manifestPath)
	if err != nil {
		return doctorInstalled(*prefix, err)
	}
	fmt.Printf("manifest=%s\n", source.Location)
	fmt.Printf("platform=%s\n", source.Document.Platform)
	fmt.Printf("channel=%s\n", source.Document.Channel)
	if artifact, err := pm.AndroidPrefixArtifact(source); err == nil {
		fmt.Printf("artifact=%s\n", artifact.Artifact.Name)
		fmt.Printf("version=%s\n", artifact.Artifact.Version)
		fmt.Printf("provider=%s\n", artifact.Artifact.Metadata["provider"])
		fmt.Printf("provider_role=%s\n", artifact.Artifact.Metadata["provider_role"])
		fmt.Printf("archive_url=%s\n", artifact.URL)
		fmt.Printf("hardcoded_termux_policy=%s\n", artifact.Artifact.Metadata["hardcoded_termux_policy"])
	}
	fmt.Println("ok=true")
	return nil
}

func listAvailable(args []string) error {
	fs := commonFlagSet("list-available")
	manifestPath := fs.String("manifest", pm.DefaultAndroidDevManifestURL, "artifact manifest URL/path")
	prefix := fs.String("prefix", defaultPrefix(), "installed prefix used for offline list fallback")
	if err := fs.Parse(args); err != nil {
		return err
	}
	source, err := loadSource(*manifestPath)
	if err != nil {
		return listInstalled(*prefix, err)
	}
	for _, name := range pm.AvailablePackages(source) {
		fmt.Println(name)
	}
	return nil
}

func install(args []string) error {
	fs := commonFlagSet("install")
	manifestPath := fs.String("manifest", pm.DefaultAndroidDevManifestURL, "artifact manifest URL/path")
	prefix := fs.String("prefix", defaultPrefix(), "installation prefix (defaults to $PREFIX)")
	cacheDir := fs.String("cache-dir", pm.DefaultCacheDir(), "download/cache directory")
	args = reorderFlags(args, map[string]bool{
		"manifest":  true,
		"prefix":    true,
		"cache-dir": true,
	})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("install expects at least one package")
	}
	if strings.TrimSpace(*prefix) == "" {
		return fmt.Errorf("installation prefix is empty; set $PREFIX or pass --prefix")
	}

	source, err := loadSource(*manifestPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	for _, pkg := range fs.Args() {
		var result pm.InstallResult
		if pkg == pm.DevBaselinePackage {
			result, err = pm.InstallDevBaseline(ctx, source, *prefix, *cacheDir)
		} else {
			result, err = pm.InstallAndroidTestBinary(ctx, source, pkg, *prefix, *cacheDir)
		}
		if err != nil {
			return err
		}
		fmt.Printf("installed=%s\n", result.Package)
		fmt.Printf("prefix=%s\n", result.Prefix)
		fmt.Printf("provider=%s\n", result.Provider)
		fmt.Printf("version=%s\n", result.Version)
		fmt.Printf("files=%d\n", result.FileCount)
		fmt.Printf("dirs=%d\n", result.DirCount)
		fmt.Printf("symlinks=%d\n", result.SymlinkCount)
	}
	return nil
}

func commonFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ExitOnError)
}

func doctorInstalled(prefix string, manifestErr error) error {
	stamp, err := pm.LoadInstallStamp(prefix)
	if err != nil {
		return fmt.Errorf("manifest unavailable (%v) and no install stamp at prefix %q: %w", manifestErr, prefix, err)
	}
	fmt.Printf("howl_pm_host_platform=%s\n", pm.CurrentHostPlatform())
	fmt.Printf("manifest=%s\n", stamp.Manifest)
	fmt.Printf("installed=true\n")
	fmt.Printf("package=%s\n", stamp.Package)
	fmt.Printf("artifact=%s\n", stamp.Artifact)
	fmt.Printf("version=%s\n", stamp.Version)
	fmt.Printf("provider=%s\n", stamp.Provider)
	fmt.Printf("prefix=%s\n", prefix)
	fmt.Printf("files=%d\n", stamp.Files)
	fmt.Printf("symlinks=%d\n", stamp.Symlinks)
	fmt.Println("ok=true")
	return nil
}

func listInstalled(prefix string, manifestErr error) error {
	stamp, err := pm.LoadInstallStamp(prefix)
	if err != nil {
		return fmt.Errorf("manifest unavailable (%v) and no install stamp at prefix %q: %w", manifestErr, prefix, err)
	}
	fmt.Println(stamp.Package)
	return nil
}

func pkgListInstalled(args []string) error {
	fs := commonFlagSet("pkg-list-installed")
	prefix := fs.String("prefix", defaultPrefix(), "installed prefix")
	if err := fs.Parse(args); err != nil {
		return err
	}
	stamp, err := pm.LoadInstallStamp(*prefix)
	if err != nil {
		return fmt.Errorf("no install stamp at prefix %q: %w", *prefix, err)
	}
	fmt.Println(stamp.Package)
	return nil
}

func pkgSearch(args []string) error {
	fs := commonFlagSet("pkg-search")
	manifestPath := fs.String("manifest", pm.DefaultAndroidDevManifestURL, "artifact manifest URL/path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("pkg search expects exactly one pattern")
	}
	pattern := strings.ToLower(fs.Arg(0))
	source, err := loadSource(*manifestPath)
	if err != nil {
		return err
	}
	for _, name := range pm.AvailablePackages(source) {
		if strings.Contains(strings.ToLower(name), pattern) {
			fmt.Println(name)
		}
	}
	return nil
}

func pkgShow(args []string) error {
	fs := commonFlagSet("pkg-show")
	manifestPath := fs.String("manifest", pm.DefaultAndroidDevManifestURL, "artifact manifest URL/path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("pkg show expects exactly one package")
	}
	name := fs.Arg(0)
	source, err := loadSource(*manifestPath)
	if err != nil {
		return err
	}
	found := false
	for _, pkg := range pm.AvailablePackages(source) {
		if pkg == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("package %q not found", name)
	}
	fmt.Printf("package=%s\n", name)
	for _, artifact := range source.Document.Artifacts {
		if artifact.Name == name {
			fmt.Printf("version=%s\n", artifact.Version)
			fmt.Printf("provider=%s\n", artifact.Metadata["provider"])
			fmt.Printf("kind=%s\n", artifact.Kind)
			return nil
		}
	}
	return nil
}

func defaultPrefix() string {
	if value := os.Getenv("PREFIX"); value != "" {
		return value
	}
	return ""
}

func reorderFlags(args []string, takesValue map[string]bool) []string {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if before, _, ok := strings.Cut(name, "="); ok {
			name = before
		}
		if strings.Contains(arg, "=") || !takesValue[name] {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positionals...)
}

func loadSource(location string) (pm.Source, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return pm.LoadSource(ctx, location)
}

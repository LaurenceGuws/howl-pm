package pm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

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

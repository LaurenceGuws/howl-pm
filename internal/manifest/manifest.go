// Package manifest defines and validates Howl PM artifact manifests.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/howl/howl-pm/internal/contract"
)

const SchemaVersion = 1

type Document struct {
	SchemaVersion int        `json:"schema_version"`
	Project       string     `json:"project"`
	Platform      string     `json:"platform"`
	Channel       string     `json:"channel"`
	Artifacts     []Artifact `json:"artifacts"`
	Notes         []string   `json:"notes,omitempty"`
}

type Artifact struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Version     string            `json:"version"`
	URL         string            `json:"url"`
	SHA256      string            `json:"sha256"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Limitations []string          `json:"limitations,omitempty"`
}

func NewSkeleton(platform string, channel string) (Document, error) {
	if platform != contract.PlatformAndroid && platform != contract.PlatformIOS {
		return Document{}, fmt.Errorf("unsupported platform %q", platform)
	}
	if channel == "" {
		return Document{}, fmt.Errorf("channel must not be empty")
	}

	doc := Document{
		SchemaVersion: SchemaVersion,
		Project:       contract.ProjectName,
		Platform:      platform,
		Channel:       channel,
		Artifacts:     []Artifact{},
		Notes: []string{
			"Generated skeleton only. Add pinned artifacts before consumption.",
			"Android and iOS mechanics are intentionally platform-specific.",
		},
	}

	if platform == contract.PlatformAndroid {
		doc.Artifacts = append(doc.Artifacts, Artifact{
			Name:     "howl-android-userland-bootstrap",
			Kind:     contract.ArtifactKindPrefixArchive,
			Version:  "0.0.0-dev",
			URL:      "TODO",
			SHA256:   "TODO",
			Metadata: contract.AndroidPrefixMetadata(contract.ProviderRoleDevBootstrap),
			Limitations: []string{
				"Development skeleton. Not a signed product channel.",
				"Must not point at unmodified com.termux-rooted package payloads.",
			},
		})
	}

	if platform == contract.PlatformIOS {
		doc.Artifacts = append(doc.Artifacts, Artifact{
			Name:    "howl-ios-tool-bundle",
			Kind:    contract.ArtifactKindIOSBundle,
			Version: "0.0.0-dev",
			URL:     "TODO",
			SHA256:  "TODO",
			Metadata: map[string]string{
				"execution_model": "platform-policy-pending",
			},
			Limitations: []string{
				"iOS is not an apt-like executable userland.",
				"Do not copy Android package assumptions into this platform.",
			},
		})
	}

	return doc, nil
}

func Load(path string) (Document, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	var doc Document
	if err := json.Unmarshal(bytes, &doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (doc Document) Validate() error {
	if doc.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", doc.SchemaVersion)
	}
	if doc.Project != contract.ProjectName {
		return fmt.Errorf("unexpected project %q", doc.Project)
	}
	if doc.Platform != contract.PlatformAndroid && doc.Platform != contract.PlatformIOS {
		return fmt.Errorf("unsupported platform %q", doc.Platform)
	}
	if doc.Channel == "" {
		return fmt.Errorf("channel must not be empty")
	}
	for i, artifact := range doc.Artifacts {
		if artifact.Name == "" {
			return fmt.Errorf("artifact[%d].name must not be empty", i)
		}
		if artifact.Kind == "" {
			return fmt.Errorf("artifact[%d].kind must not be empty", i)
		}
		if artifact.Version == "" {
			return fmt.Errorf("artifact[%d].version must not be empty", i)
		}
		if artifact.URL == "" {
			return fmt.Errorf("artifact[%d].url must not be empty", i)
		}
		if artifact.SHA256 == "" {
			return fmt.Errorf("artifact[%d].sha256 must not be empty", i)
		}
		if artifact.Size < 0 {
			return fmt.Errorf("artifact[%d].size must be non-negative", i)
		}
		if artifact.Kind == contract.ArtifactKindTestBinary {
			if doc.Platform != contract.PlatformAndroid {
				return fmt.Errorf("artifact[%d].kind %s is only valid for platform android", i, contract.ArtifactKindTestBinary)
			}
			rel := artifact.Metadata["install_relative_path"]
			if rel == "" {
				return fmt.Errorf("artifact[%d].metadata.install_relative_path must not be empty for %s", i, contract.ArtifactKindTestBinary)
			}
			if err := validateInstallRelativePath(rel); err != nil {
				return fmt.Errorf("artifact[%d].metadata.install_relative_path: %w", i, err)
			}
		}
		if artifact.Kind == contract.ArtifactKindPackageEntry {
			if err := validatePackageEntryMetadata(i, artifact); err != nil {
				return err
			}
		}
		if isProviderDerivedKind(artifact.Kind) {
			if err := validateProviderMetadata(i, artifact); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateInstallRelativePath(rel string) error {
	if rel == "" {
		return fmt.Errorf("path must not be empty")
	}
	if rel[0] == '/' {
		return fmt.Errorf("path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("path must name a file under the prefix")
	}
	return nil
}

func isProviderDerivedKind(kind string) bool {
	switch kind {
	case contract.ArtifactKindPrefixArchive, contract.ArtifactKindPackageIndex, contract.ArtifactKindTermuxDeb, contract.ArtifactKindTestBinary, contract.ArtifactKindPackageEntry:
		return true
	default:
		return false
	}
}

func validateProviderMetadata(index int, artifact Artifact) error {
	required := []string{
		"provider",
		"provider_role",
		"provider_platform",
		"provider_architecture",
	}
	for _, key := range required {
		if artifact.Metadata[key] == "" {
			return fmt.Errorf("artifact[%d].metadata.%s must not be empty for %s", index, key, artifact.Kind)
		}
	}
	return nil
}

func validatePackageEntryMetadata(index int, artifact Artifact) error {
	visibility := artifact.Metadata["visibility"]
	switch visibility {
	case "public", "private":
	default:
		return fmt.Errorf("artifact[%d].metadata.visibility must be public or private for %s", index, contract.ArtifactKindPackageEntry)
	}

	strategy := artifact.Metadata["install_strategy"]
	switch strategy {
	case "prefix-archive":
		if artifact.Metadata["artifact_ref"] == "" {
			return fmt.Errorf("artifact[%d].metadata.artifact_ref must not be empty for prefix-archive package entry", index)
		}
	case "termux-package":
		if artifact.Metadata["source_package"] == "" {
			return fmt.Errorf("artifact[%d].metadata.source_package must not be empty for termux-package package entry", index)
		}
		if artifact.Metadata["source_index_ref"] == "" {
			return fmt.Errorf("artifact[%d].metadata.source_index_ref must not be empty for termux-package package entry", index)
		}
	case contract.ArtifactKindTestBinary:
		if artifact.Metadata["artifact_ref"] == "" {
			return fmt.Errorf("artifact[%d].metadata.artifact_ref must not be empty for %s package entry", index, contract.ArtifactKindTestBinary)
		}
	default:
		return fmt.Errorf("artifact[%d].metadata.install_strategy %q is unsupported for %s", index, strategy, contract.ArtifactKindPackageEntry)
	}
	return nil
}

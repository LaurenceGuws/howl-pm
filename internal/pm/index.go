package pm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/howl/howl-pm/internal/androidrepo"
	"github.com/howl/howl-pm/internal/manifest"
)

func loadIndexFromArtifact(ctx context.Context, source Source, artifact manifest.Artifact) (androidrepo.Index, error) {
	artifactURL, err := ResolveURL(source.Location, artifact.URL)
	if err != nil {
		return androidrepo.Index{}, err
	}
	var payload []byte
	if IsURL(artifactURL) {
		payload, err = readURL(ctx, artifactURL)
	} else {
		payload, err = os.ReadFile(filepath.Clean(artifactURL))
	}
	if err != nil {
		return androidrepo.Index{}, err
	}
	if artifact.Size >= 0 && int64(len(payload)) != artifact.Size {
		return androidrepo.Index{}, fmt.Errorf("index artifact size mismatch for %s", artifact.Name)
	}
	if got := androidrepo.HashBytes(payload); got != artifact.SHA256 {
		return androidrepo.Index{}, fmt.Errorf("index artifact sha256 mismatch for %s", artifact.Name)
	}
	return androidrepo.ParseIndex(strings.NewReader(string(payload)))
}

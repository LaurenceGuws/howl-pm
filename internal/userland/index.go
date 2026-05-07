package userland

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/howl/howl-pm/internal/manifest"
	"github.com/howl/howl-pm/internal/termux"
)

func loadIndexFromArtifact(ctx context.Context, source Source, artifact manifest.Artifact) (termux.Index, error) {
	artifactURL, err := ResolveURL(source.Location, artifact.URL)
	if err != nil {
		return termux.Index{}, err
	}
	var payload []byte
	if IsURL(artifactURL) {
		payload, err = readURL(ctx, artifactURL)
	} else {
		payload, err = os.ReadFile(filepath.Clean(artifactURL))
	}
	if err != nil {
		return termux.Index{}, err
	}
	if artifact.Size >= 0 && int64(len(payload)) != artifact.Size {
		return termux.Index{}, fmt.Errorf("index artifact size mismatch for %s", artifact.Name)
	}
	if got := termux.HashBytes(payload); got != artifact.SHA256 {
		return termux.Index{}, fmt.Errorf("index artifact sha256 mismatch for %s", artifact.Name)
	}
	return termux.ParseIndex(strings.NewReader(string(payload)))
}

package userland

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/howl/howl-pm/internal/manifest"
)

func LoadSource(ctx context.Context, location string) (Source, error) {
	if strings.TrimSpace(location) == "" {
		return Source{}, errors.New("manifest location must not be empty")
	}

	var payload []byte
	var err error
	if IsURL(location) {
		payload, err = readURL(ctx, location)
	} else {
		payload, err = os.ReadFile(filepath.Clean(location))
	}
	if err != nil {
		return Source{}, err
	}

	var doc manifest.Document
	if err := json.Unmarshal(payload, &doc); err != nil {
		return Source{}, err
	}
	if err := doc.Validate(); err != nil {
		return Source{}, err
	}
	return Source{Location: location, Document: doc}, nil
}

func ResolveURL(base string, value string) (string, error) {
	if IsURL(value) {
		return value, nil
	}
	if IsURL(base) {
		parsedBase, err := url.Parse(base)
		if err != nil {
			return "", err
		}
		parsedValue, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		return parsedBase.ResolveReference(parsedValue).String(), nil
	}
	return filepath.Join(filepath.Dir(base), value), nil
}

func IsURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func DefaultCacheDir() string {
	if value := os.Getenv("HOWL_PM_CACHE"); value != "" {
		return value
	}
	if value := os.Getenv("ZIDE_PM_CACHE"); value != "" {
		return value
	}
	if value := os.Getenv("XDG_CACHE_HOME"); value != "" {
		return filepath.Join(value, "howl-pm")
	}
	if runtime.GOOS == "android" {
		if value := os.Getenv("TMPDIR"); value != "" {
			return filepath.Join(value, "howl-pm-cache")
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cache", "howl-pm")
	}
	return filepath.Join(os.TempDir(), "howl-pm-cache")
}

package userland

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func readURL(ctx context.Context, target string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	addAuthHeaders(request)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", target, response.Status)
	}
	return io.ReadAll(response.Body)
}

func downloadURL(ctx context.Context, target string, outputPath string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	addAuthHeaders(request)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", target, response.Status)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Body)
	return err
}

func addAuthHeaders(request *http.Request) {
	request.Header.Set("User-Agent", "howl-pm")
	if request.URL.Host != "github.com" {
		return
	}
	for _, name := range []string{"HOWL_PM_GITHUB_TOKEN", "ZIDE_PM_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if token := os.Getenv(name); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
			return
		}
	}
	if token := ghAuthToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
}

func ghAuthToken() string {
	gh, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	output, err := exec.Command(gh, "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func verifyPath(path string, wantSize int64, wantSHA256 string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if wantSize >= 0 && info.Size() != wantSize {
		return false
	}
	got, err := sha256Path(path)
	return err == nil && got == wantSHA256
}

func sha256Path(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

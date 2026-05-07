package userland

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	pkgprefix "github.com/howl/howl-pm/internal/prefix"
)

type extractStats struct {
	files    int
	dirs     int
	symlinks int
}

func ExtractUSRToPrefix(archivePath string, prefixRoot string) (extractStats, error) {
	prefixRoot = filepath.Clean(prefixRoot)
	if err := os.MkdirAll(prefixRoot, 0o755); err != nil {
		return extractStats{}, err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return extractStats{}, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return extractStats{}, err
	}
	defer gzipReader.Close()

	var stats extractStats
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, err
		}
		relative, ok := strings.CutPrefix(filepath.ToSlash(filepath.Clean(header.Name)), "usr/")
		if !ok || relative == "" {
			continue
		}
		target, err := safeJoin(prefixRoot, relative)
		if err != nil {
			return stats, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)&0o777); err != nil {
				return stats, err
			}
			stats.dirs++
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return stats, err
			}
			if err := writeRegularFile(target, reader, os.FileMode(header.Mode)&0o777); err != nil {
				return stats, err
			}
			stats.files++
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return stats, err
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return stats, err
			}
			stats.symlinks++
		}
	}
	return stats, nil
}

func copyFile(source string, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func writeRegularFile(path string, reader io.Reader, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func safeJoin(root string, relative string) (string, error) {
	clean := filepath.Clean(relative)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path %q", relative)
	}
	target := filepath.Join(root, clean)
	resolvedRelative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if resolvedRelative == ".." || strings.HasPrefix(resolvedRelative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive path escapes prefix: %q", relative)
	}
	return target, nil
}

func installDebIntoPrefix(debPath string, stagingRoot string, prefixRoot string) (extractStats, error) {
	if err := os.RemoveAll(stagingRoot); err != nil {
		return extractStats{}, err
	}
	if err := os.MkdirAll(stagingRoot, 0o755); err != nil {
		return extractStats{}, err
	}
	extracted, err := pkgprefix.ExtractDebUSR(debPath, stagingRoot)
	if err != nil {
		return extractStats{}, err
	}
	merged, err := mergeTree(filepath.Join(stagingRoot, "usr"), prefixRoot)
	if err != nil {
		return extractStats{}, err
	}
	_ = extracted
	return merged, nil
}

func mergeTree(sourceRoot string, targetRoot string) (extractStats, error) {
	var stats extractStats
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return stats, err
	}
	err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target, err := safeJoin(targetRoot, relative)
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(target, mode.Perm()); err != nil {
				return err
			}
			stats.dirs++
		case mode&os.ModeSymlink != 0:
			linkname, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(linkname, target); err != nil {
				return err
			}
			stats.symlinks++
		case mode.IsRegular():
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			err = writeRegularFile(target, src, mode.Perm())
			_ = src.Close()
			if err != nil {
				return err
			}
			stats.files++
		}
		return nil
	})
	return stats, err
}

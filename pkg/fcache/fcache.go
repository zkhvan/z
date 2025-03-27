package fcache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/zkhvan/z/pkg/oslib"
)

var ErrNotFound = errors.New("cache not found")

func NormalizeCacheDir(cacheDir string) string {
	if cacheDir != "" {
		return cacheDir
	}

	cacheDir = oslib.Expand("~/.cache")
	if os.Getenv("XDG_CACHE_DIR") != "" {
		cacheDir = os.Getenv("XDG_CACHE_DIR")
	}

	return filepath.Join(cacheDir, "z")
}

func LoadMany[T any](dir, key string) ([]T, error) {
	now := time.Now().Unix()

	_, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}

	var latestFile string
	var latestTimestamp int64

	pattern := fmt.Sprintf("%s-%%d.json", key)
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		var timestamp int64
		if _, err := fmt.Sscanf(d.Name(), pattern, &timestamp); err != nil {
			return nil
		}

		if timestamp < now {
			// File is expired, skip it
			return nil
		}

		if timestamp > latestTimestamp {
			latestTimestamp = timestamp
			latestFile = path
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if latestFile == "" {
		return nil, ErrNotFound
	}

	file, err := root.OpenFile(latestFile, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data []T
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func SaveMany[T any](dir, key string, data []T, expiry time.Time) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-%d.json", key, expiry.Unix())
	file, err := root.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(data); err != nil {
		return err
	}

	if err := cleanupOldCacheFiles(root, key, expiry); err != nil {
		return fmt.Errorf("failed to cleanup old cache files: %w", err)
	}

	return nil
}

func cleanupOldCacheFiles(root *os.Root, key string, latestTimestamp time.Time) error {
	pattern := fmt.Sprintf("%s-%%d.json", key)
	err := fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(d.Name())
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		var timestamp int64
		if _, err := fmt.Sscanf(d.Name(), pattern, &timestamp); err != nil {
			return nil
		}

		if timestamp < latestTimestamp.Unix() {
			if err := root.Remove(path); err != nil {
				return fmt.Errorf("failed to remove expired cache file %q: %w", path, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to cleanup old cache files: %w", err)
	}

	return nil
}

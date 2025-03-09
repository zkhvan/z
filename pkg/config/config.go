package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/zkhvan/z/pkg/cmdutil"
)

var _ cmdutil.Config = (*provider)(nil)

type provider struct {
	k *koanf.Koanf
}

// List implements cmdutil.Config.
func (p *provider) List() string {
	return p.k.Sprint()
}

func (p *provider) Bool(path string) bool {
	return p.k.Bool(path)
}

func (p *provider) BoolMap(path string) map[string]bool {
	return p.k.BoolMap(path)
}

func (p *provider) Bools(path string) []bool {
	return p.k.Bools(path)
}

func (p *provider) Bytes(path string) []byte {
	return p.k.Bytes(path)
}

func (p *provider) Duration(path string) time.Duration {
	return p.k.Duration(path)
}

func (p *provider) Float64(path string) float64 {
	return p.k.Float64(path)
}

func (p *provider) Float64Map(path string) map[string]float64 {
	return p.k.Float64Map(path)
}

func (p *provider) Float64s(path string) []float64 {
	return p.k.Float64s(path)
}

func (p *provider) Int(path string) int {
	return p.k.Int(path)
}

func (p *provider) Int64Map(path string) map[string]int64 {
	return p.k.Int64Map(path)
}

func (p *provider) Int64s(path string) []int64 {
	return p.k.Int64s(path)
}

func (p *provider) IntMap(path string) map[string]int {
	return p.k.IntMap(path)
}

func (p *provider) Ints(path string) []int {
	return p.k.Ints(path)
}

func (p *provider) String(path string) string {
	return p.k.String(path)
}

func (p *provider) StringMap(path string) map[string]string {
	return p.k.StringMap(path)
}

func (p *provider) Strings(path string) []string {
	return p.k.Strings(path)
}

func (p *provider) StringsMap(path string) map[string][]string {
	return p.k.StringsMap(path)
}

func (p *provider) Time(path, layout string) time.Time {
	return p.k.Time(path, layout)
}

func (p *provider) Int64(path string) int64 {
	return p.k.Int64(path)
}

func (p *provider) Get(path string) any {
	return p.k.Get(path)
}

func (p *provider) Unmarshal(key string, v interface{}) error {
	return p.k.UnmarshalWithConf(key, v, koanf.UnmarshalConf{
		Tag: "json",
	})
}

func New() (cmdutil.Config, error) {
	k := koanf.New(".")

	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "config.yaml")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, err
	}

	return &provider{k: k}, nil
}

// userConfigDir returns the default root directory to use for user-specific
// configuration data. It mimics os.UserConfigDir(), but overrides the
// defaults for darwin to respect XDG specifications.
func userConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "darwin":
		dir = os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
			}
			dir += "/Library/Application Support"
		} else if !filepath.IsAbs(dir) {
			return "", errors.New("path in $XDG_CONFIG_HOME is relative")
		}

	default:
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return dir, err
		}
	}

	return dir, nil
}

func configDir() (string, error) {
	baseDir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("error detecting user configuration directory: %w", err)
	}

	return filepath.Join(baseDir, "z"), nil
}

package project

import (
	"fmt"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/fcache"
)

type Service struct {
	cfg Config

	refreshCache bool
	cacheDir     string
}

type ServiceOption func(*Service)

func WithRefreshCache(refreshCache bool) ServiceOption {
	return func(s *Service) {
		s.refreshCache = refreshCache
	}
}

func WithCacheDir(cacheDir string) ServiceOption {
	return func(s *Service) {
		s.cacheDir = fcache.NormalizeCacheDir(cacheDir)
	}
}

func NewService(config cmdutil.Config, opts ...ServiceOption) (*Service, error) {
	cfg, err := NewConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating config: %w", err)
	}

	s := &Service{cfg: cfg}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

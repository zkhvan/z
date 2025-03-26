package project

import (
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

func NewService(cfg Config, opts ...ServiceOption) *Service {
	cfg = cfg.setDefaults()

	s := &Service{cfg: cfg}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

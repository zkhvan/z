package project

import (
	"fmt"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/exec"
	"github.com/zkhvan/z/pkg/fcache"
	"github.com/zkhvan/z/pkg/gh"
)

var defaultExecutor exec.Interface = exec.New()

type Service struct {
	cfg      Config
	executor exec.Interface
	gh       *gh.Client

	refreshCache bool
	cacheDir     string
}

type ServiceOption func(*Service)

func WithExecutor(executor exec.Interface) ServiceOption {
	return func(s *Service) {
		s.executor = executor
		s.gh.SetExecutor(executor)
	}
}

func WithGHClient(client *gh.Client) ServiceOption {
	return func(s *Service) {
		s.gh = client
	}
}

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

	s := &Service{
		cfg:      cfg,
		executor: defaultExecutor,
		gh:       gh.NewClient(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

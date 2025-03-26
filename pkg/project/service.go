package project

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	cfg = cfg.setDefaults()

	return &Service{cfg: cfg}
}

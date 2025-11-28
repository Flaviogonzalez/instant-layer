package defaults

import "github.com/flaviogonzalez/instant-layer/internal/types"

type TemplateFactory func(...Option) *types.Service
type Option func(*types.Service)

func applyOptions(s *types.Service, opts ...Option) *types.Service {
	for _, o := range opts {
		o(s)
	}
	return s
}

func WithName(name string) Option {
	return func(s *types.Service) {
		s.Name = name
	}
}

func WithPort(port int) Option {
	return func(s *types.Service) {
		s.Port = port
	}
}

func WithPostgres() Option {
	return func(s *types.Service) {
		s.DB = &types.Database{
			Driver:      "pgx",
			Port:        5432,
			TimeoutConn: 10,
		}
		s.Packages = append(s.Packages,
			&types.Package{Name: "config", Files: []*types.File{DefaultConfigFile(s)}},
		)
	}
}

func WithMain() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages,
			&types.Package{
				Name:  "cmd",
				Files: []*types.File{DefaultMainFile(s)},
			},
		)
	}
}

func WithRoutes() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages,
			&types.Package{
				Name:  "routes",
				Files: []*types.File{DefaultRoutesFile(s)},
			},
		)
	}
}

func WithHandlers() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages, DefaultHandlersPackage(s))
	}
}

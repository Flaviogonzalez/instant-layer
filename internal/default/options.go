package defaults

import service "github.com/flaviogonzalez/instant-layer/internal/services"

type TemplateFactory func(...Option) *service.Service
type Option func(*service.Service)

func applyOptions(s *service.Service, opts ...Option) *service.Service {
	for _, o := range opts {
		o(s)
	}
	return s
}

func WithName(name string) Option {
	return func(s *service.Service) {
		s.Name = name
	}
}

func WithPort(port int) Option {
	return func(s *service.Service) {
		s.Port = port
	}
}

func WithPostgres() Option {
	return func(s *service.Service) {
		s.DB = &service.Database{
			Driver:      "pgx",
			Port:        5432,
			TimeoutConn: 10,
		}
		s.Packages = append(s.Packages,
			&service.Package{Name: "config", Files: []*service.File{DefaultConfigFile(s)}},
		)
	}
}

func WithMain() Option {
	return func(s *service.Service) {
		s.Packages = append(s.Packages,
			&service.Package{
				Name:  "cmd",
				Files: []*service.File{DefaultMainFile(s)},
			},
		)
	}
}

func WithRoutes() Option {
	return func(s *service.Service) {
		s.Packages = append(s.Packages,
			&service.Package{
				Name:  "routes",
				Files: []*service.File{DefaultRoutesFile(s)},
			},
		)
	}
}

func WithHandlers() Option {
	return func(s *service.Service) {
		s.Packages = append(s.Packages, DefaultHandlersPackage(s))
	}
}

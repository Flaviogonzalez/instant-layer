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

// WithBrokerEvent adds the broker event package (emitter + event)
func WithBrokerEvent() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages, BrokerEventPackage(s))
	}
}

// WithListenerEvent adds the listener event package (consumer + event)
func WithListenerEvent() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages, ListenerEventPackage(s))
	}
}

// WithBrokerConfig adds the config for broker service (no DB, with RabbitMQ)
func WithBrokerConfig() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages,
			&types.Package{Name: "config", Files: []*types.File{BrokerConfigFile(s)}},
		)
	}
}

// WithListenerConfig adds the config for listener service (no HTTP server)
func WithListenerConfig() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages,
			&types.Package{Name: "config", Files: []*types.File{ListenerConfigFile(s)}},
		)
	}
}

// WithBrokerMain adds main.go for broker service
func WithBrokerMain() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages,
			&types.Package{
				Name:  "cmd",
				Files: []*types.File{BrokerMainFile(s)},
			},
		)
	}
}

// WithListenerMain adds main.go for listener service
func WithListenerMain() Option {
	return func(s *types.Service) {
		s.Packages = append(s.Packages,
			&types.Package{
				Name:  "cmd",
				Files: []*types.File{ListenerMainFile(s)},
			},
		)
	}
}

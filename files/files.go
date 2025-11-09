package files

import (
	"go/ast"
	"instant-layer/factory"
)

type ConnectionType int

const (
	GRPC ConnectionType = iota
	JSON
	RPC
)

type EventType int

const (
	RabbitMQ EventType = iota
	Kafka
)

type DBType int

const (
	MySQL DBType = iota
	PGX
)

type GenConfig struct {
	Services          []Service
	PackageGenerators ServiceMap
}

type ConfigOption func(*GenConfig)
type ServiceMap map[string][]func(service *Service, config *GenConfig) *ast.File

var defaultPackageGenerators ServiceMap = ServiceMap{
	"config":   {ConfigFile},
	"routes":   {RoutesFile},
	"main":     {MainFile},
	"helpers":  {},
	"handlers": {},
	"models":   {},
}

func NewGenConfig(options ...ConfigOption) *GenConfig {
	pkgGens := make(ServiceMap)

	for key, value := range defaultPackageGenerators {
		newSlice := make([]func(service *Service, config *GenConfig) *ast.File, len(value))
		copy(newSlice, value)
		pkgGens[key] = newSlice
	}

	config := &GenConfig{
		Services:          make([]Service, 0),
		PackageGenerators: pkgGens,
	}

	// 2. it will apply for each config
	for _, option := range options {
		option(config)
	}

	return config
}

type Service struct {
	Connections []*Connection
	Packages    []*Package
	Name        string // e.g ecommerce-service. defined by the user.
	Routes      []*Route
}

func WithService(service Service) ConfigOption {
	return func(c *GenConfig) {
		c.Services = append(c.Services, service)
	}
}

func WithPackage(name string, gens ...func(*Service, *GenConfig) *ast.File) ConfigOption {
	return func(c *GenConfig) {
		c.PackageGenerators[name] = gens
	}
}

func AddToPackage(name string, gens ...func(*Service, *GenConfig) *ast.File) ConfigOption {
	return func(c *GenConfig) {
		if _, ok := c.PackageGenerators[name]; !ok {
			c.PackageGenerators[name] = make([]func(*Service, *GenConfig) *ast.File, 0)
		}
		c.PackageGenerators[name] = append(c.PackageGenerators[name], gens...)
	}
}

func WithNewPackage(name string, gens ...func(*Service, *GenConfig) *ast.File) ConfigOption {
	return WithPackage(name, gens...)
}

type Package struct {
	Name  string
	Files []File
}

type File struct {
	Name string
	Data ast.File
}

// config of Connection through services
type EventDriven struct {
}

// every route in connection will send data through listener and broker with the same DataType
type Connection struct {
	Route Route
}

// this info would be passed down by File, to generate the specified content
type Route struct {
	Path    string
	Method  string // POST, DELETE, PUT, GET.
	Handler Handler
}

type Handler struct {
	Name           string // this would be the name of the route + handler.
	AttachedModels []Model
}

type Model struct {
	Name   string // e.g. "User"
	Fields []Field
}

type Field struct {
	Name string // e.g. "ID"
	Type string // e.g. "int"
	Json string // e.g. "id"
}

func (app *GenConfig) GetServices() []string {
	services := make([]string, 0, len(app.Services))
	for _, v := range app.Services {
		services = append(services, v.Name)
	}
	return services
}

func CollectImports(usedPackages map[string]string) *ast.GenDecl {
	imports := make([]*ast.ImportSpec, 0)
	for path, alias := range usedPackages {
		imports = append(imports, factory.NewImport(path, alias))
	}
	return factory.NewImportDecl(imports...)
}

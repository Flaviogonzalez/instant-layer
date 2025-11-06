package files

import "go/ast"

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

type FilesConfig struct {
	Services []Service
}

type Service struct {
	Connections []*Connection
	Packages    []*Package
	Name        string // e.g ecommerce-service. defined by the user.
	Routes      []*Route
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

func (app *FilesConfig) GetServices() []string {
	services := make([]string, 0, len(app.Services))
	for _, v := range app.Services {
		services = append(services, v.Name)
	}
	return services
}

type ServiceMap map[string][]func(service *Service) *ast.File

// predefined data
var packages ServiceMap = ServiceMap{
	"config":   {ConfigFile},
	"routes":   {RoutesFile},
	"main":     {MainFile},
	"handlers": {},
	"models":   {},
}

func (app *Service) GetPackages() ServiceMap {
	return packages
}

package files

import (
	"go/ast"
)

// global config

type GenConfig struct {
	Services          []Service
	PackageGenerators ServiceMap
}

type Config struct {
	Name      string
	GenConfig GenConfig
}

type Service struct {
	ServerType Server
	Packages   []*Package
	Name       string // e.g ecommerce-service. defined by the user.
}

type Package struct {
	Name  string
	Files []File
}

type File struct {
	Name string
	Data *ast.File
}

// Server Connections
type API struct {
	Routes []*Route
}

type Broker struct {
	Routes []*Route
}

type Listener struct {
	Routes []*Route
}

func (*API) srv()      {}
func (*Broker) srv()   {}
func (*Listener) srv() {}

type Server interface {
	srv()
}

type Route struct {
	Path    string // /user
	Method  string // POST, DELETE, PUT, GET.
	Handler Handler
}

type Handler struct {
	Name           string // this would be the name of the route + handler.
	AttachedModels []Model
	// more config below
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

type ConfigOption func(*GenConfig)

type ServiceMap map[string][]func(service *Service, config *GenConfig) *File

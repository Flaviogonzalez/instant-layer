package files

import (
	"go/ast"
)

// ---------------------------------------------------------------------
// Types (order matters – ServiceMap must be known before it is used)
// ---------------------------------------------------------------------

type ServiceMap map[string][]func(*Service, *GenConfig) *File

type GenConfig struct {
	Services          []Service
	PackageGenerators ServiceMap
}

type Config struct {
	Name      string
	GenConfig *GenConfig
}

type Service struct {
	ServerType Server
	Packages   []*Package
	Name       string // e.g. ecommerce-service
}

type Package struct {
	Name  string
	Files []File
}

type File struct {
	Name string
	Data *ast.File
}

// ---------------------------------------------------------------------
// Server hierarchy (interface + concrete types)
// ---------------------------------------------------------------------

type Server interface {
	srv()
}

type API struct{ Routes []*Route }
type Broker struct{ Routes []*Route }
type Listener struct{ Routes []*Route }

func (*API) srv()      {}
func (*Broker) srv()   {}
func (*Listener) srv() {}

// ---------------------------------------------------------------------
// Route / Handler / Model
// ---------------------------------------------------------------------

type Route struct {
	Path    string // /user
	Method  string // POST, DELETE, PUT, GET
	Handler Handler
}

type Handler struct {
	Name           string
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

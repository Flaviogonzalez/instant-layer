package files

import (
	"go/ast"
)

// ---------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------
type ServiceMap map[string][]func(*Service, *GenConfig) *File

type GenConfig struct {
	Services          []Service `json:"services,omitempty"`
	PackageGenerators ServiceMap
}

type Config struct {
	Name      string     `json:"name,omitempty"`
	GenConfig *GenConfig `json:"genConfig,omitempty"`
}

type Service struct {
	ServerType Server     `json:"serverType,omitempty"`
	Packages   []*Package `json:"packages,omitempty"`
	Name       string     `json:"name,omitempty"` // e.g. ecommerce-service
	Port       int        `json:"port,omitempty"`
}

type Package struct {
	Name  string `json:"name,omitempty"`
	Files []File `json:"files,omitempty"`
}

type File struct {
	Name string    `json:"name,omitempty"`
	Data *ast.File `json:"data,omitempty"`
}

// ---------------------------------------------------------------------
// Server hierarchy (interface + concrete types)
// ---------------------------------------------------------------------
type Server interface {
	srv()
}

type API struct {
	DB           Database     `json:"db,omitzero"`
	RoutesConfig RoutesConfig `json:"routesConfig,omitzero"`
}

func (*API) srv() {}

// ---------------------------------------------------------------------
// concrete DB config instances for each type of srv (interface + concrete types)
// ---------------------------------------------------------------------
type Database struct {
	TimeoutConn int    `json:"timeoutConn,omitempty"`
	Driver      string `json:"driver,omitempty"`
	URL         string `json:"url,omitempty"` // always in .ENV, need to create a .env file with DATABASE_URL={{url}}
}

// ---------------------------------------------------------------------
// Route
// ---------------------------------------------------------------------
type RoutesConfig struct {
	CORS        CorsOptions    `json:"cors,omitzero"`
	RoutesGroup []*RoutesGroup `json:"routesGroup,omitempty"`
}

type CorsOptions struct {
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedMethods   []string `json:"allowedMethods,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	AllowCredentials bool     `json:"allowCredentials,omitempty"`
	MaxAge           int      `json:"maxAge,omitempty"`
}

type RoutesGroup struct {
	Prefix     string     `json:"prefix,omitempty"`
	Middleware Middleware `json:"middleware,omitzero"` // todo: omitempty when middleware fulfill
	Routes     []*Route   `json:"routes,omitempty"`
}

type Middleware struct {
}

type Route struct {
	Path    string `json:"path,omitempty"`    // /user
	Method  string `json:"method,omitempty"`  // POST, DELETE, PUT, GET
	Handler string `json:"handler,omitempty"` // User Implementation.
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

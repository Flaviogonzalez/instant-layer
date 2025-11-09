package files

import (
	"go/ast"
)

// ---------------------------------------------------------------------
// Types
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
	Port       int
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

type API struct {
	DB           Database
	RoutesConfig RoutesConfig
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

// ---------------------------------------------------------------------
// concrete DB config instances for each type of srv (interface + concrete types)
// ---------------------------------------------------------------------

type Database struct {
	TimeoutConn int
	Driver      string
	URL         string      // always in .ENV, need to create a .env file with DATABASE_URL={{url}}
	DataTypes   []*Override // SQLC configuration || maybe using Models to Generate schemas.sql and Routes to generate queries.sql
}

type Override struct {
	DBType   string // DB (MYSQL OR PGX)_PREFIX or COLUMN
	GOType   GoType
	Tag      GoTag
	Unsigned bool
	Nullable bool
}

type GoTag struct {
	Type  string // json, bson, etc
	value string // value
}

type GoType struct {
	Import  string
	Package string
	Type    string
	Pointer bool
	Slice   bool
}

// ---------------------------------------------------------------------
// Route / Handler / Model
// ---------------------------------------------------------------------

type RoutesConfig struct {
	CORS        CorsOptions
	RoutesGroup []*RoutesGroup
}

type CorsOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type RoutesGroup struct {
	prefix     string
	Middleware Middleware
	Routes     []*Route
}

type Middleware struct {
}

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

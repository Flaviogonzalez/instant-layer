package types

import (
	"go/ast"
	"time"
)

// Service represents a microservice to be generated
type Service struct {
	Packages     []*Package    `json:"-"`
	Name         string        `json:"name,omitempty"` // e.g. ecommerce-service
	Port         int           `json:"port,omitempty"`
	DB           *Database     `json:"db,omitzero"`
	RoutesConfig *RoutesConfig `json:"routesConfig,omitzero"`
	Benchmark    *Benchmark    `json:"benchmark,omitempty"`
}

// Package represents a Go package to be generated
type Package struct {
	Name  string // package name
	Files []*File
}

// File represents a Go source file to be generated
type File struct {
	Name    string // filename
	Content *ast.File
}

// Database configuration
type Database struct {
	TimeoutConn int    `json:"timeoutConn,omitempty"`
	Driver      string `json:"driver,omitempty"`
	URL         string `json:"url,omitempty"` // always in .ENV, need to create a .env file with DATABASE_URL={{url}}
	Port        int
}

// RoutesConfig holds the routing configuration for a service
type RoutesConfig struct {
	CORS        *CorsOptions   `json:"cors,omitzero"`
	RoutesGroup []*RoutesGroup `json:"routesGroup,omitempty"`
}

// CorsOptions for CORS middleware configuration
type CorsOptions struct {
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedMethods   []string `json:"allowedMethods,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	AllowCredentials bool     `json:"allowCredentials,omitempty"`
	MaxAge           int      `json:"maxAge,omitempty"`
}

// RoutesGroup represents a group of routes with shared configuration
type RoutesGroup struct {
	Prefix     string     `json:"prefix,omitempty"`
	Middleware Middleware `json:"middleware,omitzero"` // todo: omitempty when middleware fulfill
	Routes     []*Route   `json:"routes,omitempty"`
}

// Middleware configuration (placeholder for future expansion)
type Middleware struct {
}

// Route represents a single HTTP route
type Route struct {
	Path    string `json:"path,omitempty"`    // /user
	Method  string `json:"method,omitempty"`  // POST, DELETE, PUT, GET
	Handler string `json:"handler,omitempty"` // User Implementation.
}

// ConnectionProtocol represents the protocol used for inter-service communication
type ConnectionProtocol string

const (
	ProtocolHTTP      ConnectionProtocol = "http"
	ProtocolHTTPS     ConnectionProtocol = "https"
	ProtocolGRPC      ConnectionProtocol = "grpc"
	ProtocolWebSocket ConnectionProtocol = "ws"
	ProtocolRPC       ConnectionProtocol = "rpc"
)

// Connection represents a connection from one service to another
type Connection struct {
	FromService string             `json:"fromService"`      // source service name
	ToService   string             `json:"toService"`        // target service name (e.g., "auth-service")
	Protocol    ConnectionProtocol `json:"protocol"`         // http, grpc, ws, rpc
	Route       string             `json:"route,omitempty"`  // target route (e.g., "/login")
	Method      string             `json:"method,omitempty"` // HTTP method if applicable
	SourceFile  string             `json:"sourceFile"`       // file where connection was found
	SourceLine  int                `json:"sourceLine"`       // line number
	Valid       bool               `json:"valid"`            // whether target service/route exists
	Error       string             `json:"error,omitempty"`  // validation error if any
}

// Benchmark holds performance metrics for a service
type Benchmark struct {
	ServiceName   string        `json:"serviceName"`
	TotalFiles    int           `json:"totalFiles"`
	TotalLines    int           `json:"totalLines"`
	TotalPackages int           `json:"totalPackages"`
	HasTests      bool          `json:"hasTests"`
	TestFiles     int           `json:"testFiles"`
	BuildTime     time.Duration `json:"buildTime,omitempty"`
	BinarySize    int64         `json:"binarySize,omitempty"`
	Dependencies  int           `json:"dependencies"`
	LastAnalyzed  time.Time     `json:"lastAnalyzed"`
}

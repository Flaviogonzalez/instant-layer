package service

import (
	"go/ast"

	"github.com/flaviogonzalez/instant-layer/internal/routes"
)

type Service struct {
	Packages     []*Package           `json:"-"`
	Name         string               `json:"name,omitempty"` // e.g. ecommerce-service
	Port         int                  `json:"port,omitempty"`
	DB           *Database            `json:"db,omitzero"`
	RoutesConfig *routes.RoutesConfig `json:"routesConfig,omitzero"`
}

type Package struct {
	Name  string // package name
	Files []*File
}

type File struct {
	Name    string // filename
	Content *ast.File
}

type Database struct {
	TimeoutConn int    `json:"timeoutConn,omitempty"`
	Driver      string `json:"driver,omitempty"`
	URL         string `json:"url,omitempty"` // always in .ENV, need to create a .env file with DATABASE_URL={{url}}
	Port        int
}

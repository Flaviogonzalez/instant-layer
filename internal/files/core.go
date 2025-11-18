package files

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"path/filepath"
)

type ConfigOption func(*GenConfig)

func WithService(s Service) ConfigOption {
	return func(c *GenConfig) { c.Services = append(c.Services, s) }
}

// ---------------------------------------------------------------------
// Default generators
// ---------------------------------------------------------------------

var defaultPackageGenerators = ServiceMap{
	"config":     {ConfigFile},
	"routes":     {RoutesFile},
	"main":       {MainFile},
	"helpers":    {},
	"handlers":   {},
	"models":     {},
	"middleware": {},
}

// ---------------------------------------------------------------------
// Public factory
// ---------------------------------------------------------------------

func NewGenConfig(options ...ConfigOption) *GenConfig {
	pkgGens := make(ServiceMap)
	for k, v := range defaultPackageGenerators {
		pkgGens[k] = append([]func(*Service, *GenConfig) *File(nil), v...)
	}

	cfg := &GenConfig{
		Services:          make([]Service, 0),
		PackageGenerators: pkgGens,
	}

	for _, opt := range options {
		opt(cfg)
	}

	cfg.loadHandlers()
	cfg.generateServices()
	return cfg
}

// ---------------------------------------------------------------------
// Core generation
// ---------------------------------------------------------------------

func (c *GenConfig) generateServices() {
	for i := range c.Services {
		svc := &c.Services[i]
		svc.Packages = make([]*Package, 0, len(c.PackageGenerators))

		for pkgName, gens := range c.PackageGenerators {
			pkg := &Package{
				Name:  pkgName,
				Files: make([]File, 0, len(gens)),
			}

			for _, gen := range gens {
				if gen == nil {
					continue
				}
				if f := gen(svc, c); f != nil {
					pkg.Files = append(pkg.Files, *f)
				}
			}

			if len(pkg.Files) > 0 {
				svc.Packages = append(svc.Packages, pkg)
			}
		}
	}
}

func (c *GenConfig) loadHandlers() {
	for _, s := range c.Services {
		if _, ok := s.ServerType.(*API); ok {
			api := s.ServerType.(*API)

			// for each routegroup
			for _, group := range api.RoutesConfig.RoutesGroup {
				for _, route := range group.Routes {
					handler := route.Handler
					c.PackageGenerators["handlers"] = append(c.PackageGenerators["handlers"], func(s *Service, c *GenConfig) *File {
						return createHandlerFile(handler)
					})
				}
			}
		}
	}
}

// ---------------------------------------------------------------------
// Project initialisation (writes files to disk)
// ---------------------------------------------------------------------

func (c *Config) InitGeneration(outputDir string) {
	c.GenConfig.loadHandlers()

	fset := token.NewFileSet()
	projectRoot := filepath.Join(outputDir, c.Name)

	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		log.Fatalf("Error creando proyecto: %v", err)
	}

	for _, svc := range c.GenConfig.Services {
		svcPath := filepath.Join(projectRoot, svc.Name)
		if err := os.MkdirAll(svcPath, 0755); err != nil {
			log.Fatalf("Error creando servicio '%s': %v", svc.Name, err)
		}

		writeGoMod(svcPath, fmt.Sprintf("%s", c.Name))

		// Packages
		for _, pkg := range svc.Packages {
			pkgPath := filepath.Join(svcPath, pkg.Name)
			if pkg.Name != "main" {
				if err := os.MkdirAll(pkgPath, 0755); err != nil {
					log.Fatalf("Error creando paquete '%s': %v", pkg.Name, err)
				}
			}

			for _, file := range pkg.Files {
				fileName := file.Name
				filePath := filepath.Join(pkgPath, fileName)

				// main package → always "main.go" at service root
				if pkg.Name == "main" {
					fileName = "main.go"
					filePath = filepath.Join(svcPath, fileName)
				}

				writeASTFile(fset, filePath, file.Data)
			}
		}
	}

	log.Printf("Proyecto generado en: %s\n", projectRoot)
}

// ---------------------------------------------------------------------
// Helper writers
// ---------------------------------------------------------------------

func writeASTFile(fset *token.FileSet, path string, node ast.Node) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		log.Fatalf("Error formateando AST (%s): %v", path, err)
	}
	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("Error en format.Source (%s): %v", path, err)
	}
	if err := os.WriteFile(path, pretty, 0644); err != nil {
		log.Fatalf("Error escribiendo %s: %v", path, err)
	}
}

func writeGoMod(dir, modulePath string) {
	content := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/jackc/pgx/v5   v5.5.0
	github.com/lib/pq         v1.10.9
)
`, modulePath)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644); err != nil {
		log.Fatalf("Error escribiendo go.mod (%s): %v", dir, err)
	}
}

// ---------------------------------------------------------------------
// Utility (list service names)
// ---------------------------------------------------------------------

func (c *GenConfig) GetServices() []string {
	out := make([]string, 0, len(c.Services))
	for _, s := range c.Services {
		out = append(out, s.Name)
	}
	return out
}

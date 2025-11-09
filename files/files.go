package files

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"instant-layer/factory"
	"log"
	"os"
	"path/filepath"
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

type Config struct {
	Name      string
	GenConfig GenConfig
}

func (c *Config) InitGeneration(outputDir, projectName string) {
	fs := token.NewFileSet()
	projectRoot := filepath.Join(outputDir, projectName)

	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		log.Fatalf("Error creando proyecto: %v", err)
	}

	for _, service := range c.GenConfig.Services {
		servicePath := filepath.Join(projectRoot, service.Name)
		if err := os.MkdirAll(servicePath, 0755); err != nil {
			log.Fatalf("Error creando servicio '%s': %v", service.Name, err)
		}

		// go.mod
		writeGoMod(servicePath, service.Name)

		// Paquetes
		for packageName, genFuncs := range c.GenConfig.PackageGenerators {
			pkgPath := filepath.Join(servicePath, packageName)
			if packageName != "main" {
				if err := os.MkdirAll(pkgPath, 0755); err != nil {
					log.Fatalf("Error creando paquete '%s': %v", packageName, err)
				}
			}

			for _, genFunc := range genFuncs {
				var fileName string
				if packageName == "main" {
					fileName = "main.go"
				} else {
					// here will check for multiple files
					// notice that if multiple files has the same name it will throw an error.
					// i don't expect that each handler will stand like {}_handler or {}_handler1, 2
					fileName = fmt.Sprintf("%s.go", packageName)
				}

				filePath := filepath.Join(pkgPath, fileName)
				if packageName == "main" {
					filePath = filepath.Join(servicePath, "main.go")
				}

				writeFile(fs, filePath, genFunc(&service, &c.GenConfig).Data)
			}
		}
	}

	log.Printf("Proyecto generado en: %s\n", projectRoot)
}

func writeFile(fs *token.FileSet, path string, node ast.Node) {
	var buf bytes.Buffer

	if err := format.Node(&buf, fs, node); err != nil {
		log.Fatal("Error formateando código:", err)
	}

	prettyCode, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal("Error en format.Source (embellecimiento):", err)
	}

	f, err := os.Create(path)
	if err != nil {
		log.Fatal("Error creando archivo:", err)
	}
	defer f.Close()

	if _, err := f.Write(prettyCode); err != nil {
		log.Fatal("Error escribiendo archivo final:", err)
	}
}

func writeGoMod(servicePath, modulePath string) {
	content := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/jackc/pgx/v5 v5.5.0
	github.com/lib/pq v1.10.9
)
`, modulePath)

	if err := os.WriteFile(filepath.Join(servicePath, "go.mod"), []byte(content), 0644); err != nil {
		log.Fatal("Error escribiendo go.mod:", err)
	}
}

type ConfigOption func(*GenConfig)
type ServiceMap map[string][]func(service *Service, config *GenConfig) *File

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
		newSlice := make([]func(service *Service, config *GenConfig) *File, len(value))
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

	config.GenerateServices()
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

func WithPackage(name string, gens ...func(*Service, *GenConfig) *File) ConfigOption {
	return func(c *GenConfig) {
		c.PackageGenerators[name] = gens
	}
}

func AddToPackage(name string, gens ...func(*Service, *GenConfig) *File) ConfigOption {
	return func(c *GenConfig) {
		if _, ok := c.PackageGenerators[name]; !ok {
			c.PackageGenerators[name] = make([]func(*Service, *GenConfig) *File, 0)
		}
		c.PackageGenerators[name] = append(c.PackageGenerators[name], gens...)
	}
}

func WithNewPackage(name string, gens ...func(*Service, *GenConfig) *File) ConfigOption {
	return WithPackage(name, gens...)
}

func (c *GenConfig) GenerateServices() {
	for i := range c.Services {
		service := &c.Services[i]

		service.Packages = make([]*Package, 0)

		for pkgName, generators := range c.PackageGenerators {
			newPkg := &Package{Name: pkgName, Files: make([]File, 0)}

			for _, genFunc := range generators {
				if genFunc == nil {
					continue // will skip if no generator is found
				}

				generatedFile := genFunc(service, c)

				if generatedFile != nil {
					newPkg.Files = append(newPkg.Files, *generatedFile)
				}
			}

			if len(newPkg.Files) > 0 {
				service.Packages = append(service.Packages, newPkg)
			}
		}
	}
}

type Package struct {
	Name  string
	Files []File
}

type File struct {
	Name string
	Data *ast.File
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

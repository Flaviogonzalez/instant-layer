package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"instant-layer/files"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Name        string
	FilesConfig files.FilesConfig
}

func main() {
	outputDir := "dist"
	projectName := "instant-layer"

	config := &Config{
		Name: projectName,
		FilesConfig: files.FilesConfig{
			Services: []files.Service{
				{Name: "article-service"},
			},
		},
	}

	config.InitGeneration(outputDir, projectName)
}

func (c *Config) InitGeneration(outputDir, projectName string) {
	fs := token.NewFileSet()
	projectRoot := filepath.Join(outputDir, projectName)

	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		log.Fatalf("Error creando proyecto: %v", err)
	}

	for _, service := range c.FilesConfig.Services {
		servicePath := filepath.Join(projectRoot, service.Name)
		if err := os.MkdirAll(servicePath, 0755); err != nil {
			log.Fatalf("Error creando servicio '%s': %v", service.Name, err)
		}

		// go.mod
		writeGoMod(servicePath, service.Name)

		// Paquetes
		for packageName, genFuncs := range service.GetPackages() {
			pkgPath := filepath.Join(servicePath, packageName)
			if packageName != "main" {
				if err := os.MkdirAll(pkgPath, 0755); err != nil {
					log.Fatalf("Error creando paquete '%s': %v", packageName, err)
				}
			}

			for i, genFunc := range genFuncs {
				var fileName string
				if packageName == "main" {
					fileName = "main.go"
				} else {
					fileName = fmt.Sprintf("%s_file%d.go", packageName, i)
				}

				filePath := filepath.Join(pkgPath, fileName)
				if packageName == "main" {
					filePath = filepath.Join(servicePath, "main.go")
				}

				writeFile(fs, filePath, genFunc(&service))
			}
		}
	}

	log.Printf("Proyecto generado en: %s\n", projectRoot)
}

func writeFile(fs *token.FileSet, path string, node ast.Node) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal("Error creando archivo:", err)
	}
	defer f.Close()

	if err := format.Node(f, fs, node); err != nil {
		log.Fatal("Error formateando código:", err)
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

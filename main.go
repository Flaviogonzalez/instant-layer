package main

import (
	"bytes"
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
	Name      string
	GenConfig files.GenConfig
}

func main() {
	outputDir := "dist"
	projectName := "instant-layer"

	config := &Config{
		Name: projectName,
		GenConfig: files.GenConfig{
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

				writeFile(fs, filePath, &genFunc(&service, &c.GenConfig).Data)
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

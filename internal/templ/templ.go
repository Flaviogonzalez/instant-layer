package templ

import (
	"embed"
	"os"
	"path/filepath"
	"text/template"

	"github.com/flaviogonzalez/instant-layer/internal/types"
)

//go:embed *.tmpl
var templates embed.FS

// GoModData holds data for generating go.mod files
type GoModData struct {
	Name         string
	GoVersion    string
	Dependencies []Dependency
}

// Dependency represents a Go module dependency
type Dependency struct {
	Path    string
	Version string
}

// DockerComposeData holds data for generating docker-compose.yml
type DockerComposeData struct {
	Name     string
	Services []*ServiceData
}

// ServiceData represents a service in docker-compose
type ServiceData struct {
	Name      string
	Port      int
	DB        *types.Database
	DependsOn []string
}

// DefaultDependencies returns the default dependencies for a new service
func DefaultDependencies() []Dependency {
	return []Dependency{
		{Path: "github.com/go-chi/chi/v5", Version: "v5.1.0"},
		{Path: "github.com/go-chi/cors", Version: "v1.2.1"},
	}
}

// DefaultGoVersion returns the latest stable Go version
func DefaultGoVersion() string {
	return "1.23"
}

// GenerateGoMod generates a go.mod file for a service
func GenerateGoMod(outputPath string, data GoModData) error {
	if data.GoVersion == "" {
		data.GoVersion = DefaultGoVersion()
	}

	tmpl, err := template.ParseFS(templates, "gomod.tmpl")
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// GenerateDockerCompose generates a docker-compose.yml file
func GenerateDockerCompose(outputPath string, data DockerComposeData) error {
	tmpl, err := template.ParseFS(templates, "dockercompose.tmpl")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// GetTemplate returns a parsed template by name
func GetTemplate(name string) (*template.Template, error) {
	return template.ParseFS(templates, name)
}

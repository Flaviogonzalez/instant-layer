package templ

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviogonzalez/instant-layer/internal/types"
)

func TestDefaultGoVersion(t *testing.T) {
	version := DefaultGoVersion()
	if version == "" {
		t.Error("DefaultGoVersion() should return a non-empty string")
	}
	// Should be a valid Go version format (e.g., "1.23")
	if !strings.Contains(version, ".") {
		t.Errorf("DefaultGoVersion() = %s; want format like '1.23'", version)
	}
}

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies()
	if len(deps) == 0 {
		t.Error("DefaultDependencies() should return at least one dependency")
	}

	// Check that chi is included
	haschi := false
	for _, dep := range deps {
		if strings.Contains(dep.Path, "chi") {
			haschi = true
			break
		}
	}
	if !haschi {
		t.Error("DefaultDependencies() should include go-chi")
	}
}

func TestGenerateGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "go.mod")

	data := GoModData{
		Name:      "test-service",
		GoVersion: "1.23",
		Dependencies: []Dependency{
			{Path: "github.com/example/pkg", Version: "v1.0.0"},
		},
	}

	err := GenerateGoMod(outputPath, data)
	if err != nil {
		t.Fatalf("GenerateGoMod() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Verify content
	contentStr := string(content)
	if !strings.Contains(contentStr, "module test-service") {
		t.Error("Generated go.mod should contain module name")
	}
	if !strings.Contains(contentStr, "go 1.23") {
		t.Error("Generated go.mod should contain go version")
	}
	if !strings.Contains(contentStr, "github.com/example/pkg") {
		t.Error("Generated go.mod should contain dependencies")
	}
}

func TestGenerateGoModWithDefaultVersion(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "go.mod")

	data := GoModData{
		Name: "test-service",
		// GoVersion intentionally empty - should use default
	}

	err := GenerateGoMod(outputPath, data)
	if err != nil {
		t.Fatalf("GenerateGoMod() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if !strings.Contains(string(content), "go "+DefaultGoVersion()) {
		t.Error("Generated go.mod should use default Go version when not specified")
	}
}

func TestGenerateDockerCompose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "docker-compose.yml")

	data := DockerComposeData{
		Name: "myproject",
		Services: []*ServiceData{
			{
				Name: "auth-service",
				Port: 8080,
				DB: &types.Database{
					URL: "postgres://localhost:5432/auth",
				},
				DependsOn: []string{"broker-service"},
			},
			{
				Name: "broker-service",
				Port: 8081,
			},
		},
	}

	err := GenerateDockerCompose(outputPath, data)
	if err != nil {
		t.Fatalf("GenerateDockerCompose() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify content
	if !strings.Contains(contentStr, "name: myproject") {
		t.Error("Generated docker-compose should contain project name")
	}
	if !strings.Contains(contentStr, "auth-service:") {
		t.Error("Generated docker-compose should contain auth-service")
	}
	if !strings.Contains(contentStr, "broker-service:") {
		t.Error("Generated docker-compose should contain broker-service")
	}
	if !strings.Contains(contentStr, "8080:8080") {
		t.Error("Generated docker-compose should contain port mapping")
	}
	if !strings.Contains(contentStr, "depends_on:") {
		t.Error("Generated docker-compose should contain depends_on")
	}
	if !strings.Contains(contentStr, "myproject-network") {
		t.Error("Generated docker-compose should contain network name")
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmplName string
		wantErr  bool
	}{
		{"valid gomod template", "gomod.tmpl", false},
		{"valid dockercompose template", "dockercompose.tmpl", false},
		{"invalid template", "nonexistent.tmpl", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := GetTemplate(tt.tmplName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tmpl == nil {
				t.Error("GetTemplate() returned nil template without error")
			}
		})
	}
}

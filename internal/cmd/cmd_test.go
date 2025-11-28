package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviogonzalez/instant-layer/internal/config"
	"github.com/flaviogonzalez/instant-layer/internal/templ"
	"github.com/flaviogonzalez/instant-layer/internal/types"
)

// TestGenerateServiceGoMod tests go.mod generation for services
func TestGenerateServiceGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	servicePath := filepath.Join(tmpDir, "test-service")

	if err := os.MkdirAll(servicePath, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	err := generateServiceGoMod(servicePath, "test-service")
	if err != nil {
		t.Fatalf("generateServiceGoMod() error = %v", err)
	}

	// Verify go.mod was created
	goModPath := filepath.Join(servicePath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	contentStr := string(content)

	// Check module name
	if !strings.Contains(contentStr, "module test-service") {
		t.Error("go.mod should contain module name")
	}

	// Check go version
	if !strings.Contains(contentStr, "go "+templ.DefaultGoVersion()) {
		t.Error("go.mod should contain Go version")
	}

	// Check dependencies
	if !strings.Contains(contentStr, "go-chi/chi") {
		t.Error("go.mod should contain chi dependency")
	}
	if !strings.Contains(contentStr, "jackc/pgx") {
		t.Error("go.mod should contain pgx dependency")
	}
}

// TestUpdateLayerWithService tests adding service to layer.json
func TestUpdateLayerWithService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial layer.json
	layerJSON := `{"name": "test-project", "root": "` + filepath.ToSlash(tmpDir) + `", "Services": []}`
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(layerJSON), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Add a service
	newService := &types.Service{
		Name: "auth-service",
		Port: 8080,
	}

	err := updateLayerWithService(tmpDir, newService)
	if err != nil {
		t.Fatalf("updateLayerWithService() error = %v", err)
	}

	// Reload and verify
	reloaded := &config.Layer{Root: tmpDir}
	if err := reloaded.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if len(reloaded.Services) != 1 {
		t.Errorf("Services = %d, want 1", len(reloaded.Services))
	}

	if reloaded.Services[0].Name != "auth-service" {
		t.Errorf("Service name = %q, want %q", reloaded.Services[0].Name, "auth-service")
	}
}

// TestRegenerateDockerCompose tests docker-compose.yml generation
func TestRegenerateDockerCompose(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layer.json with services
	layerJSON := `{
		"name": "test-project",
		"root": "` + filepath.ToSlash(tmpDir) + `",
		"Services": [
			{"name": "auth-service", "port": 8080},
			{"name": "broker-service", "port": 8081}
		]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(layerJSON), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	err := regenerateDockerCompose(tmpDir)
	if err != nil {
		t.Fatalf("regenerateDockerCompose() error = %v", err)
	}

	// Verify docker-compose.yml was created
	dcPath := filepath.Join(tmpDir, "docker-compose.yml")
	content, err := os.ReadFile(dcPath)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	contentStr := string(content)

	// Check content
	if !strings.Contains(contentStr, "name: test-project") {
		t.Error("docker-compose.yml should contain project name")
	}
	if !strings.Contains(contentStr, "auth-service:") {
		t.Error("docker-compose.yml should contain auth-service")
	}
	if !strings.Contains(contentStr, "broker-service:") {
		t.Error("docker-compose.yml should contain broker-service")
	}
	if !strings.Contains(contentStr, "8080:8080") {
		t.Error("docker-compose.yml should contain port mapping for auth-service")
	}
	if !strings.Contains(contentStr, "8081:8081") {
		t.Error("docker-compose.yml should contain port mapping for broker-service")
	}
}

// TestRegenerateDockerComposeWithHydrate tests docker-compose regeneration using Hydrate
func TestRegenerateDockerComposeWithHydrate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layer.json with empty services (will use Hydrate)
	layerJSON := `{
		"name": "hydrate-project",
		"root": "` + filepath.ToSlash(tmpDir) + `",
		"Services": []
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(layerJSON), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Create a service directory with go.mod
	svcPath := filepath.Join(tmpDir, "api-service")
	if err := os.MkdirAll(svcPath, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	goMod := "module api-service\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(svcPath, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	err := regenerateDockerCompose(tmpDir)
	if err != nil {
		t.Fatalf("regenerateDockerCompose() error = %v", err)
	}

	// Verify docker-compose.yml was created
	dcPath := filepath.Join(tmpDir, "docker-compose.yml")
	content, err := os.ReadFile(dcPath)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	contentStr := string(content)

	// Should contain the hydrated service
	if !strings.Contains(contentStr, "api-service:") {
		t.Error("docker-compose.yml should contain hydrated api-service")
	}
}

// TestGenerateEmptyDockerCompose tests empty docker-compose generation
func TestGenerateEmptyDockerCompose(t *testing.T) {
	tmpDir := t.TempDir()

	layer := &config.Layer{
		Name:     "empty-project",
		Root:     tmpDir,
		Services: []*types.Service{},
	}

	err := generateEmptyDockerCompose(layer)
	if err != nil {
		t.Fatalf("generateEmptyDockerCompose() error = %v", err)
	}

	// Verify docker-compose.yml was created
	dcPath := filepath.Join(tmpDir, "docker-compose.yml")
	content, err := os.ReadFile(dcPath)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	contentStr := string(content)

	// Check content
	if !strings.Contains(contentStr, "name: empty-project") {
		t.Error("docker-compose.yml should contain project name")
	}
	if !strings.Contains(contentStr, "networks:") {
		t.Error("docker-compose.yml should contain networks section")
	}
}

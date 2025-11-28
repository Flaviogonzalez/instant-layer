package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flaviogonzalez/instant-layer/internal/types"
)

// TestParseModuleName tests go.mod parsing
func TestParseModuleName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple module",
			content:  "module test-service\n\ngo 1.23",
			expected: "test-service",
		},
		{
			name:     "github module",
			content:  "module github.com/user/myservice\n\ngo 1.23",
			expected: "github.com/user/myservice",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "no module line",
			content:  "go 1.23\nrequire test v1.0.0",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseModuleName(tt.content)
			if result != tt.expected {
				t.Errorf("parseModuleName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestCountLines tests line counting functionality
func TestCountLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	count, err := countLines(testFile)
	if err != nil {
		t.Fatalf("countLines() error = %v", err)
	}

	// Content has 5 lines
	if count != 5 {
		t.Errorf("countLines() = %d, want 5", count)
	}
}

// TestCountLinesNonExistent tests error handling for missing files
func TestCountLinesNonExistent(t *testing.T) {
	_, err := countLines("/nonexistent/path/file.go")
	if err == nil {
		t.Error("countLines() should return error for nonexistent file")
	}
}

// TestParseProtocol tests protocol parsing
func TestParseProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected types.ConnectionProtocol
	}{
		{"http", types.ProtocolHTTP},
		{"https", types.ProtocolHTTPS},
		{"grpc", types.ProtocolGRPC},
		{"ws", types.ProtocolWebSocket},
		{"wss", types.ProtocolWebSocket},
		{"rpc", types.ProtocolRPC},
		{"unknown", types.ProtocolHTTP}, // defaults to HTTP
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseProtocol(tt.input)
			if result != tt.expected {
				t.Errorf("parseProtocol(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// createTestService creates a test service directory structure
func createTestService(t *testing.T, rootDir, serviceName string) string {
	servicePath := filepath.Join(rootDir, serviceName)
	if err := os.MkdirAll(servicePath, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	// Create go.mod
	goMod := "module " + serviceName + "\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(servicePath, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	return servicePath
}

// TestScanServices tests service discovery
func TestScanServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layer.json
	layerJSON := `{"name": "test-project", "root": "` + filepath.ToSlash(tmpDir) + `"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(layerJSON), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Create test services
	createTestService(t, tmpDir, "auth-service")
	createTestService(t, tmpDir, "broker-service")

	// Create a non-service directory (no go.mod)
	nonServiceDir := filepath.Join(tmpDir, "not-a-service")
	if err := os.MkdirAll(nonServiceDir, 0755); err != nil {
		t.Fatalf("Failed to create non-service dir: %v", err)
	}

	layer := &Layer{
		Name: "test-project",
		Root: tmpDir,
	}

	services, err := layer.ScanServices()
	if err != nil {
		t.Fatalf("ScanServices() error = %v", err)
	}

	if len(services) != 2 {
		t.Errorf("ScanServices() returned %d services, want 2", len(services))
	}

	// Verify service names
	names := make(map[string]bool)
	for _, svc := range services {
		names[svc.Name] = true
	}

	if !names["auth-service"] || !names["broker-service"] {
		t.Errorf("ScanServices() missing expected services, got: %v", names)
	}
}

// TestScanRoutesFile tests route parsing
func TestScanRoutesFile(t *testing.T) {
	tmpDir := t.TempDir()

	routesDir := filepath.Join(tmpDir, "routes")
	if err := os.MkdirAll(routesDir, 0755); err != nil {
		t.Fatalf("Failed to create routes dir: %v", err)
	}

	routesContent := `package routes

import "github.com/go-chi/chi/v5"

func NewRoutes(mux *chi.Mux) {
	mux.Post("/login", handlers.Login)
	mux.Get("/user/{id}", handlers.GetUser)
	mux.Put("/user/{id}", handlers.UpdateUser)
	mux.Delete("/user/{id}", handlers.DeleteUser)
}
`
	if err := os.WriteFile(filepath.Join(routesDir, "routes.go"), []byte(routesContent), 0644); err != nil {
		t.Fatalf("Failed to write routes.go: %v", err)
	}

	layer := &Layer{Root: tmpDir}
	config := layer.scanRoutes(tmpDir)

	if config == nil {
		t.Fatal("scanRoutes() returned nil")
	}

	if len(config.RoutesGroup) == 0 {
		t.Fatal("scanRoutes() returned empty RoutesGroup")
	}

	routes := config.RoutesGroup[0].Routes
	if len(routes) != 4 {
		t.Errorf("scanRoutes() found %d routes, want 4", len(routes))
	}

	// Verify methods
	methodCounts := make(map[string]int)
	for _, route := range routes {
		methodCounts[route.Method]++
	}

	if methodCounts["POST"] != 1 || methodCounts["GET"] != 1 || methodCounts["PUT"] != 1 || methodCounts["DELETE"] != 1 {
		t.Errorf("scanRoutes() method counts = %v, want POST:1, GET:1, PUT:1, DELETE:1", methodCounts)
	}
}

// TestHydrate tests full service hydration
func TestHydrate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layer.json
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Create auth-service with routes
	authPath := createTestService(t, tmpDir, "auth-service")
	authRoutesDir := filepath.Join(authPath, "routes")
	if err := os.MkdirAll(authRoutesDir, 0755); err != nil {
		t.Fatalf("Failed to create routes dir: %v", err)
	}

	routesContent := `package routes

func NewRoutes(mux *chi.Mux) {
	mux.Post("/login", handlers.Login)
}
`
	if err := os.WriteFile(filepath.Join(authRoutesDir, "routes.go"), []byte(routesContent), 0644); err != nil {
		t.Fatalf("Failed to write routes.go: %v", err)
	}

	layer := &Layer{
		Name: "test-project",
		Root: tmpDir,
	}

	err := layer.Hydrate()
	if err != nil {
		t.Fatalf("Hydrate() error = %v", err)
	}

	if len(layer.Services) != 1 {
		t.Errorf("Hydrate() found %d services, want 1", len(layer.Services))
	}

	svc := layer.Services[0]
	if svc.Name != "auth-service" {
		t.Errorf("Service name = %q, want %q", svc.Name, "auth-service")
	}

	if svc.RoutesConfig == nil {
		t.Error("Service should have RoutesConfig after hydration")
	}

	if svc.Benchmark == nil {
		t.Error("Service should have Benchmark after hydration")
	}
}

// TestCountDependencies tests dependency counting
func TestCountDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	goModContent := `module test-service

go 1.23

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/jackc/pgx/v5 v5.6.0
)

require github.com/example/single v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	layer := &Layer{}
	count := layer.countDependencies(tmpDir)

	// Should count 3: 2 in require block + 1 single require
	if count != 3 {
		t.Errorf("countDependencies() = %d, want 3", count)
	}
}

// TestFindLayerRoot tests layer.json discovery
func TestFindLayerRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "project", "services", "auth")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	// Create layer.json at project level
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.WriteFile(filepath.Join(projectDir, "layer.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Search from deeply nested directory
	root, err := FindLayerRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindLayerRoot() error = %v", err)
	}

	if root != projectDir {
		t.Errorf("FindLayerRoot() = %q, want %q", root, projectDir)
	}
}

// TestFindLayerRootNotFound tests error when layer.json doesn't exist
func TestFindLayerRootNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindLayerRoot(tmpDir)
	if err == nil {
		t.Error("FindLayerRoot() should return error when layer.json not found")
	}
}

// TestScanConnections tests inter-service connection detection
func TestScanConnections(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layer.json
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Create auth-service
	authPath := createTestService(t, tmpDir, "auth-service")

	// Create broker-service with routes
	brokerPath := createTestService(t, tmpDir, "broker-service")
	brokerRoutesDir := filepath.Join(brokerPath, "routes")
	if err := os.MkdirAll(brokerRoutesDir, 0755); err != nil {
		t.Fatalf("Failed to create routes dir: %v", err)
	}

	brokerRoutes := `package routes

func NewRoutes(mux *chi.Mux) {
	mux.Post("/send", handlers.Send)
}
`
	if err := os.WriteFile(filepath.Join(brokerRoutesDir, "routes.go"), []byte(brokerRoutes), 0644); err != nil {
		t.Fatalf("Failed to write routes.go: %v", err)
	}

	// Create file in auth-service that connects to broker-service
	authHandlersDir := filepath.Join(authPath, "handlers")
	if err := os.MkdirAll(authHandlersDir, 0755); err != nil {
		t.Fatalf("Failed to create handlers dir: %v", err)
	}

	handlerContent := `package handlers

func Login(w http.ResponseWriter, r *http.Request) {
	// Call broker service
	resp, err := http.Post("http://broker-service/send", "application/json", body)
}
`
	if err := os.WriteFile(filepath.Join(authHandlersDir, "login.go"), []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to write login.go: %v", err)
	}

	layer := &Layer{
		Name: "test-project",
		Root: tmpDir,
	}

	// First hydrate to get services with routes
	if err := layer.Hydrate(); err != nil {
		t.Fatalf("Hydrate() error = %v", err)
	}

	connections, err := layer.ScanConnections()
	if err != nil {
		t.Fatalf("ScanConnections() error = %v", err)
	}

	if len(connections) == 0 {
		t.Error("ScanConnections() should find at least one connection")
	}

	// Check that we found the auth -> broker connection
	found := false
	for _, conn := range connections {
		if conn.FromService == "auth-service" && conn.ToService == "broker-service" {
			found = true
			if conn.Protocol != types.ProtocolHTTP {
				t.Errorf("Connection protocol = %v, want HTTP", conn.Protocol)
			}
			if conn.Route != "/send" {
				t.Errorf("Connection route = %q, want %q", conn.Route, "/send")
			}
		}
	}

	if !found {
		t.Error("ScanConnections() should find auth-service -> broker-service connection")
	}
}

// TestValidateConnection tests connection validation
func TestValidateConnection(t *testing.T) {
	layer := &Layer{}

	// Create service map
	brokerService := &types.Service{
		Name: "broker-service",
		RoutesConfig: &types.RoutesConfig{
			RoutesGroup: []*types.RoutesGroup{
				{
					Routes: []*types.Route{
						{Method: "POST", Path: "/send"},
					},
				},
			},
		},
	}

	serviceMap := map[string]*types.Service{
		"broker-service": brokerService,
	}

	tests := []struct {
		name      string
		conn      *types.Connection
		wantValid bool
	}{
		{
			name: "valid connection to existing route",
			conn: &types.Connection{
				FromService: "auth-service",
				ToService:   "broker-service",
				Route:       "/send",
			},
			wantValid: true,
		},
		{
			name: "valid connection to service root",
			conn: &types.Connection{
				FromService: "auth-service",
				ToService:   "broker-service",
				Route:       "/",
			},
			wantValid: true,
		},
		{
			name: "invalid connection to non-existent service",
			conn: &types.Connection{
				FromService: "auth-service",
				ToService:   "unknown-service",
				Route:       "/api",
			},
			wantValid: false,
		},
		{
			name: "invalid connection to non-existent route",
			conn: &types.Connection{
				FromService: "auth-service",
				ToService:   "broker-service",
				Route:       "/nonexistent",
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layer.validateConnection(tt.conn, serviceMap)
			if tt.conn.Valid != tt.wantValid {
				t.Errorf("validateConnection() Valid = %v, want %v", tt.conn.Valid, tt.wantValid)
			}
		})
	}
}

// TestGetServiceDependencies tests dependency extraction
func TestGetServiceDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layer.json
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	// Create services
	authPath := createTestService(t, tmpDir, "auth-service")
	createTestService(t, tmpDir, "broker-service")
	createTestService(t, tmpDir, "db-service")

	// Add connections from auth-service
	handlersDir := filepath.Join(authPath, "handlers")
	if err := os.MkdirAll(handlersDir, 0755); err != nil {
		t.Fatalf("Failed to create handlers dir: %v", err)
	}

	handlerContent := `package handlers

func Login(w http.ResponseWriter, r *http.Request) {
	http.Post("http://broker-service/", "application/json", nil)
	http.Get("http://db-service/")
}
`
	if err := os.WriteFile(filepath.Join(handlersDir, "login.go"), []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to write login.go: %v", err)
	}

	layer := &Layer{
		Name: "test-project",
		Root: tmpDir,
	}

	if err := layer.Hydrate(); err != nil {
		t.Fatalf("Hydrate() error = %v", err)
	}

	deps := layer.GetServiceDependencies("auth-service")

	// Should find both broker-service and db-service as dependencies
	depMap := make(map[string]bool)
	for _, d := range deps {
		depMap[d] = true
	}

	if !depMap["broker-service"] || !depMap["db-service"] {
		t.Errorf("GetServiceDependencies() = %v, want broker-service and db-service", deps)
	}
}

// TestAnalyzeServiceBenchmark tests benchmark analysis
func TestAnalyzeServiceBenchmark(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service with multiple files
	svcPath := createTestService(t, tmpDir, "test-service")

	// Add handlers directory with Go files
	handlersDir := filepath.Join(svcPath, "handlers")
	if err := os.MkdirAll(handlersDir, 0755); err != nil {
		t.Fatalf("Failed to create handlers dir: %v", err)
	}

	handler1 := `package handlers

func Handler1() {}
`
	handler2 := `package handlers

func Handler2() {}
`
	testFile := `package handlers

func TestHandler1(t *testing.T) {}
`

	if err := os.WriteFile(filepath.Join(handlersDir, "handler1.go"), []byte(handler1), 0644); err != nil {
		t.Fatalf("Failed to write handler1.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(handlersDir, "handler2.go"), []byte(handler2), 0644); err != nil {
		t.Fatalf("Failed to write handler2.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(handlersDir, "handler1_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to write handler1_test.go: %v", err)
	}

	layer := &Layer{Root: tmpDir}
	svc := &types.Service{Name: "test-service"}

	benchmark := layer.analyzeServiceBenchmark(svcPath, svc)

	if benchmark == nil {
		t.Fatal("analyzeServiceBenchmark() returned nil")
	}

	if benchmark.ServiceName != "test-service" {
		t.Errorf("ServiceName = %q, want %q", benchmark.ServiceName, "test-service")
	}

	if benchmark.TotalFiles < 3 {
		t.Errorf("TotalFiles = %d, want at least 3", benchmark.TotalFiles)
	}

	if !benchmark.HasTests {
		t.Error("HasTests should be true")
	}

	if benchmark.TestFiles < 1 {
		t.Errorf("TestFiles = %d, want at least 1", benchmark.TestFiles)
	}
}

// TestLayerSave tests saving layer configuration
func TestLayerSave(t *testing.T) {
	tmpDir := t.TempDir()
	layerRoot := filepath.Join(tmpDir, "new-project")

	layer := &Layer{
		Name: "new-project",
		Root: layerRoot,
		Services: []*types.Service{
			{Name: "auth-service"},
		},
	}

	err := layer.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify layer.json was created
	layerPath := filepath.Join(layerRoot, "layer.json")
	if _, err := os.Stat(layerPath); os.IsNotExist(err) {
		t.Error("Save() should create layer.json file")
	}
}

// TestLayerUpdate tests updating layer configuration
func TestLayerUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	// First create the layer
	layer := &Layer{
		Name: "test-project",
		Root: tmpDir,
		Services: []*types.Service{
			{Name: "auth-service", Port: 8080},
		},
	}

	// Write initial layer.json
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("Failed to write initial layer.json: %v", err)
	}

	// Update should work
	err := layer.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Reload and verify
	reloaded := &Layer{Root: tmpDir}
	if err := reloaded.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if reloaded.Name != "test-project" {
		t.Errorf("Name = %q, want %q", reloaded.Name, "test-project")
	}

	if len(reloaded.Services) != 1 {
		t.Errorf("Services = %d, want 1", len(reloaded.Services))
	}
}

// TestLayerReload tests reloading layer configuration
func TestLayerReload(t *testing.T) {
	tmpDir := t.TempDir()

	layerJSON := `{
  "name": "test-project",
  "root": "` + filepath.ToSlash(tmpDir) + `"
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "layer.json"), []byte(layerJSON), 0644); err != nil {
		t.Fatalf("Failed to write layer.json: %v", err)
	}

	layer := &Layer{Root: tmpDir}
	err := layer.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if layer.Name != "test-project" {
		t.Errorf("Name = %q, want %q", layer.Name, "test-project")
	}
}

// TestConnectionPatternMatching tests the connection regex pattern
func TestConnectionPatternMatching(t *testing.T) {
	tests := []struct {
		input    string
		expected int // expected number of matches
	}{
		{`http.Get("http://auth-service/login")`, 1},
		{`http.Post("https://broker-service:8080/send", nil)`, 1},
		{`grpc.Dial("grpc://user-service/users")`, 1},
		{`ws.Connect("ws://realtime-service/events")`, 1},
		{`"rpc://payment-service/process"`, 1},
		{`http.Get("http://localhost:8080/test")`, 1}, // localhost match
		{`http.Get("http://external.com/api")`, 1},    // external domain - also matches (pattern is general)
		{`no url here`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := connectionPattern.FindAllStringSubmatch(tt.input, -1)
			if len(matches) != tt.expected {
				t.Errorf("Pattern matches = %d, want %d for input: %s", len(matches), tt.expected, tt.input)
			}
		})
	}
}

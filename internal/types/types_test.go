package types

import (
	"encoding/json"
	"testing"
)

// TestServiceJSONMarshaling tests Service serialization
func TestServiceJSONMarshaling(t *testing.T) {
	svc := &Service{
		Name: "test-service",
		Port: 8080,
		DB: &Database{
			Driver: "pgx",
			Port:   5432,
			URL:    "postgres://localhost/test",
		},
		RoutesConfig: &RoutesConfig{
			RoutesGroup: []*RoutesGroup{
				{
					Prefix: "/api",
					Routes: []*Route{
						{Method: "POST", Path: "/login", Handler: "Login"},
					},
				},
			},
		},
	}

	// Marshal
	data, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("Failed to marshal Service: %v", err)
	}

	// Unmarshal
	var decoded Service
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Service: %v", err)
	}

	// Verify
	if decoded.Name != svc.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, svc.Name)
	}
	if decoded.Port != svc.Port {
		t.Errorf("Port = %d, want %d", decoded.Port, svc.Port)
	}
	if decoded.DB == nil {
		t.Error("DB should not be nil after unmarshaling")
	}
}

// TestServicePackagesNotSerialized verifies Packages field is not serialized
func TestServicePackagesNotSerialized(t *testing.T) {
	svc := &Service{
		Name: "test-service",
		Packages: []*Package{
			{Name: "handlers"},
		},
	}

	data, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("Failed to marshal Service: %v", err)
	}

	// Packages should not appear in JSON (has `json:"-"` tag)
	jsonStr := string(data)
	if contains(jsonStr, "handlers") {
		t.Error("Packages should not be serialized to JSON")
	}
}

// TestDatabaseConfig tests Database configuration
func TestDatabaseConfig(t *testing.T) {
	db := &Database{
		Driver:      "pgx",
		Port:        5432,
		URL:         "postgres://user:pass@localhost:5432/mydb",
		TimeoutConn: 10,
	}

	if db.Driver != "pgx" {
		t.Errorf("Driver = %q, want %q", db.Driver, "pgx")
	}
	if db.Port != 5432 {
		t.Errorf("Port = %d, want %d", db.Port, 5432)
	}
}

// TestRoutesConfig tests RoutesConfig structure
func TestRoutesConfig(t *testing.T) {
	config := &RoutesConfig{
		CORS: &CorsOptions{
			AllowedOrigins:   []string{"http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowCredentials: true,
			MaxAge:           300,
		},
		RoutesGroup: []*RoutesGroup{
			{
				Prefix: "/api/v1",
				Routes: []*Route{
					{Method: "GET", Path: "/users", Handler: "GetUsers"},
					{Method: "POST", Path: "/users", Handler: "CreateUser"},
				},
			},
		},
	}

	if config.CORS == nil {
		t.Error("CORS should not be nil")
	}
	if len(config.CORS.AllowedOrigins) != 1 {
		t.Errorf("AllowedOrigins = %d, want 1", len(config.CORS.AllowedOrigins))
	}
	if len(config.RoutesGroup) != 1 {
		t.Errorf("RoutesGroup = %d, want 1", len(config.RoutesGroup))
	}
	if len(config.RoutesGroup[0].Routes) != 2 {
		t.Errorf("Routes = %d, want 2", len(config.RoutesGroup[0].Routes))
	}
}

// TestRoutesGroupPrefix tests route group prefix handling
func TestRoutesGroupPrefix(t *testing.T) {
	group := &RoutesGroup{
		Prefix: "/api/v1",
		Routes: []*Route{
			{Method: "GET", Path: "/users"},
		},
	}

	// Full path would be /api/v1/users
	fullPath := group.Prefix + group.Routes[0].Path
	if fullPath != "/api/v1/users" {
		t.Errorf("Full path = %q, want %q", fullPath, "/api/v1/users")
	}
}

// TestRouteValues tests Route field values
func TestRouteValues(t *testing.T) {
	routes := []*Route{
		{Method: "POST", Path: "/login", Handler: "Login"},
		{Method: "GET", Path: "/users/{id}", Handler: "GetUser"},
		{Method: "PUT", Path: "/users/{id}", Handler: "UpdateUser"},
		{Method: "DELETE", Path: "/users/{id}", Handler: "DeleteUser"},
		{Method: "PATCH", Path: "/users/{id}", Handler: "PatchUser"},
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true, "OPTIONS": true, "HEAD": true,
	}

	for _, route := range routes {
		if !validMethods[route.Method] {
			t.Errorf("Invalid method: %s", route.Method)
		}
		if route.Path == "" {
			t.Error("Path should not be empty")
		}
		if route.Handler == "" {
			t.Error("Handler should not be empty")
		}
	}
}

// TestConnectionProtocols tests ConnectionProtocol constants
func TestConnectionProtocols(t *testing.T) {
	tests := []struct {
		protocol ConnectionProtocol
		expected string
	}{
		{ProtocolHTTP, "http"},
		{ProtocolHTTPS, "https"},
		{ProtocolGRPC, "grpc"},
		{ProtocolWebSocket, "ws"},
		{ProtocolRPC, "rpc"},
	}

	for _, tt := range tests {
		if string(tt.protocol) != tt.expected {
			t.Errorf("Protocol %v = %q, want %q", tt.protocol, string(tt.protocol), tt.expected)
		}
	}
}

// TestConnection tests Connection structure
func TestConnection(t *testing.T) {
	conn := &Connection{
		FromService: "auth-service",
		ToService:   "broker-service",
		Protocol:    ProtocolHTTP,
		Route:       "/send",
		Method:      "POST",
		SourceFile:  "handlers/login.go",
		SourceLine:  42,
		Valid:       true,
	}

	if conn.FromService != "auth-service" {
		t.Errorf("FromService = %q, want %q", conn.FromService, "auth-service")
	}
	if conn.Protocol != ProtocolHTTP {
		t.Errorf("Protocol = %v, want %v", conn.Protocol, ProtocolHTTP)
	}
	if !conn.Valid {
		t.Error("Connection should be valid")
	}
}

// TestConnectionInvalid tests invalid Connection
func TestConnectionInvalid(t *testing.T) {
	conn := &Connection{
		FromService: "auth-service",
		ToService:   "unknown-service",
		Protocol:    ProtocolHTTP,
		Route:       "/api",
		Valid:       false,
		Error:       "target service 'unknown-service' does not exist",
	}

	if conn.Valid {
		t.Error("Connection should be invalid")
	}
	if conn.Error == "" {
		t.Error("Invalid connection should have an error message")
	}
}

// TestBenchmark tests Benchmark structure
func TestBenchmark(t *testing.T) {
	benchmark := &Benchmark{
		ServiceName:   "test-service",
		TotalFiles:    15,
		TotalLines:    1200,
		TotalPackages: 4,
		HasTests:      true,
		TestFiles:     3,
		Dependencies:  5,
	}

	if benchmark.ServiceName != "test-service" {
		t.Errorf("ServiceName = %q, want %q", benchmark.ServiceName, "test-service")
	}
	if benchmark.TotalFiles != 15 {
		t.Errorf("TotalFiles = %d, want %d", benchmark.TotalFiles, 15)
	}
	if !benchmark.HasTests {
		t.Error("HasTests should be true")
	}
}

// TestBenchmarkJSONMarshaling tests Benchmark serialization
func TestBenchmarkJSONMarshaling(t *testing.T) {
	benchmark := &Benchmark{
		ServiceName:  "json-service",
		TotalFiles:   10,
		TotalLines:   500,
		HasTests:     true,
		Dependencies: 3,
	}

	data, err := json.Marshal(benchmark)
	if err != nil {
		t.Fatalf("Failed to marshal Benchmark: %v", err)
	}

	var decoded Benchmark
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Benchmark: %v", err)
	}

	if decoded.ServiceName != benchmark.ServiceName {
		t.Errorf("ServiceName = %q, want %q", decoded.ServiceName, benchmark.ServiceName)
	}
	if decoded.TotalFiles != benchmark.TotalFiles {
		t.Errorf("TotalFiles = %d, want %d", decoded.TotalFiles, benchmark.TotalFiles)
	}
}

// TestPackage tests Package structure
func TestPackage(t *testing.T) {
	pkg := &Package{
		Name: "handlers",
		Files: []*File{
			{Name: "login.go"},
			{Name: "logout.go"},
		},
	}

	if pkg.Name != "handlers" {
		t.Errorf("Name = %q, want %q", pkg.Name, "handlers")
	}
	if len(pkg.Files) != 2 {
		t.Errorf("Files = %d, want 2", len(pkg.Files))
	}
}

// TestFile tests File structure
func TestFile(t *testing.T) {
	file := &File{
		Name:    "config.go",
		Content: nil, // AST content would be set by factory
	}

	if file.Name != "config.go" {
		t.Errorf("Name = %q, want %q", file.Name, "config.go")
	}
}

// TestCorsOptions tests CORSConfig/CorsOptions structure
func TestCorsOptions(t *testing.T) {
	cors := &CorsOptions{
		AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}

	if len(cors.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins = %d, want 2", len(cors.AllowedOrigins))
	}
	if len(cors.AllowedMethods) != 4 {
		t.Errorf("AllowedMethods = %d, want 4", len(cors.AllowedMethods))
	}
	if !cors.AllowCredentials {
		t.Error("AllowCredentials should be true")
	}
	if cors.MaxAge != 86400 {
		t.Errorf("MaxAge = %d, want %d", cors.MaxAge, 86400)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

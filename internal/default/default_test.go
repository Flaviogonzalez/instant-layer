package defaults

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/flaviogonzalez/instant-layer/internal/types"
)

// renderAST renders an AST file node to string for testing.
// Returns error if the AST cannot be printed or formatted.
func renderAST(node interface{}) (string, error) {
	var buf bytes.Buffer
	fset := token.NewFileSet()

	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

// mustRenderAST renders AST and fails test on error
func mustRenderAST(t *testing.T, node interface{}) string {
	t.Helper()
	rendered, err := renderAST(node)
	if err != nil {
		t.Fatalf("Failed to render AST: %v", err)
	}
	return rendered
}

// validateGoCode parses the given Go source and returns an error if invalid
func validateGoCode(source string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", source, parser.AllErrors)
	return err
}

// mustValidateGoCode validates Go source and fails test on error
func mustValidateGoCode(t *testing.T, source string) {
	t.Helper()
	if err := validateGoCode(source); err != nil {
		t.Fatalf("Generated code is invalid Go:\n%s\n\nError: %v", source, err)
	}
}

// TestDefaultService tests the DefaultService constructor
func TestDefaultService(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		wantName string
		wantPort int
	}{
		{
			name:     "default values",
			opts:     []Option{},
			wantName: "",
			wantPort: 8080,
		},
		{
			name:     "with name",
			opts:     []Option{WithName("auth-service")},
			wantName: "auth-service",
			wantPort: 8080,
		},
		{
			name:     "with name and port",
			opts:     []Option{WithName("user-service"), WithPort(9000)},
			wantName: "user-service",
			wantPort: 9000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := DefaultService(tt.opts...)

			if svc.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", svc.Name, tt.wantName)
			}
			if svc.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", svc.Port, tt.wantPort)
			}
			// Default service should have DB configured (from WithPostgres)
			if svc.DB == nil {
				t.Error("DB should be configured by default")
			}
			// Should have packages (config, cmd, routes)
			if len(svc.Packages) < 3 {
				t.Errorf("Should have at least 3 packages, got %d", len(svc.Packages))
			}
		})
	}
}

// TestDefaultServiceNameInGeneratedFiles verifies that s.Name is correctly
// available when generating files (regression test for option ordering bug)
func TestDefaultServiceNameInGeneratedFiles(t *testing.T) {
	svc := DefaultService(
		WithName("payment-service"),
		WithPort(8085),
	)

	// Find config package and check that import uses service name
	var configPkg *types.Package
	for _, pkg := range svc.Packages {
		if pkg.Name == "config" {
			configPkg = pkg
			break
		}
	}

	if configPkg == nil {
		t.Fatal("config package not found")
	}

	if len(configPkg.Files) == 0 {
		t.Fatal("config package has no files")
	}

	configFile := configPkg.Files[0]
	if configFile.Content == nil {
		t.Fatal("config file has no content")
	}

	rendered := mustRenderAST(t, configFile.Content)
	mustValidateGoCode(t, rendered)
	if !strings.Contains(rendered, "payment-service/routes") {
		t.Errorf("Config file should import payment-service/routes, got:\n%s", rendered)
	}
}

// TestDefaultConfigFile tests config.go generation
func TestDefaultConfigFile(t *testing.T) {
	svc := &types.Service{
		Name: "test-service",
		Port: 8080,
		DB: &types.Database{
			Driver: "pgx",
		},
	}

	file := DefaultConfigFile(svc)

	if file.Name != "config.go" {
		t.Errorf("File name = %q, want %q", file.Name, "config.go")
	}

	if file.Content == nil {
		t.Fatal("File content should not be nil")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	// Check expected content
	expectedParts := []string{
		"package config",
		"test-service/routes",
		"database/sql",
		"type Config struct",
		"func InitConfig",
		"func (app *Config) InitServer",
		"func openDB",
		"func connectToDB",
		"github.com/jackc/pgx", // pgx driver import
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Config file should contain %q", part)
		}
	}
}

// TestDefaultConfigFileWithoutDB tests config generation without DB
func TestDefaultConfigFileWithoutDB(t *testing.T) {
	svc := &types.Service{
		Name: "test-service",
		Port: 8080,
		DB:   nil, // No database
	}

	file := DefaultConfigFile(svc)
	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	// Should not have pgx imports
	if strings.Contains(rendered, "github.com/jackc/pgx") {
		t.Error("Config file without DB should not import pgx")
	}
}

// TestDefaultMainFile tests main.go generation
func TestDefaultMainFile(t *testing.T) {
	svc := &types.Service{
		Name: "my-service",
		Port: 8080,
	}

	file := DefaultMainFile(svc)

	if file.Name != "main.go" {
		t.Errorf("File name = %q, want %q", file.Name, "main.go")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package main",
		"my-service/config",
		"func main()",
		"config.InitConfig",
		"InitServer",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Main file should contain %q, got:\n%s", part, rendered)
		}
	}
}

// TestDefaultRoutesFile tests routes.go generation
func TestDefaultRoutesFile(t *testing.T) {
	svc := &types.Service{
		Name: "api-service",
		Port: 8080,
		RoutesConfig: &types.RoutesConfig{
			RoutesGroup: []*types.RoutesGroup{
				{
					Routes: []*types.Route{
						{Method: "POST", Path: "/login", Handler: "Login"},
						{Method: "GET", Path: "/users", Handler: "GetUsers"},
					},
				},
			},
		},
	}

	file := DefaultRoutesFile(svc)

	if file.Name != "routes.go" {
		t.Errorf("File name = %q, want %q", file.Name, "routes.go")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package routes",
		"api-service/handlers",
		"github.com/go-chi/chi/v5",
		"func Routes",
		"chi.NewRouter",
		"mux.Use",
		`mux.Post("/login"`,
		`mux.Get("/users"`,
		"handlers.Login",
		"handlers.GetUsers",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Routes file should contain %q, got:\n%s", part, rendered)
		}
	}
}

// TestDefaultRoutesFileWithCORS tests routes.go with CORS configuration
func TestDefaultRoutesFileWithCORS(t *testing.T) {
	svc := &types.Service{
		Name: "cors-service",
		Port: 8080,
		RoutesConfig: &types.RoutesConfig{
			CORS: &types.CorsOptions{
				AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
				AllowCredentials: true,
				MaxAge:           300,
			},
		},
	}

	file := DefaultRoutesFile(svc)
	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"github.com/go-chi/cors",
		"cors.Handler",
		"cors.Options",
		"AllowedOrigins",
		"AllowedMethods",
		"AllowCredentials",
		"MaxAge",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Routes file with CORS should contain %q", part)
		}
	}
}

// TestDefaultRoutesFileWithoutRoutes tests routes.go generation without routes
func TestDefaultRoutesFileWithoutRoutes(t *testing.T) {
	svc := &types.Service{
		Name: "empty-service",
		Port: 8080,
	}

	file := DefaultRoutesFile(svc)
	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	// Should still have basic structure
	if !strings.Contains(rendered, "package routes") {
		t.Error("Routes file should have package declaration")
	}
	if !strings.Contains(rendered, "chi.NewRouter") {
		t.Error("Routes file should create router")
	}
	if !strings.Contains(rendered, "return mux") {
		t.Error("Routes file should return mux")
	}

	// Should NOT have route method calls
	if strings.Contains(rendered, "mux.Post") || strings.Contains(rendered, "mux.Get") {
		t.Error("Routes file without routes should not have route method calls")
	}
}

// TestDefaultHandlersPackage tests handlers package generation
func TestDefaultHandlersPackage(t *testing.T) {
	svc := &types.Service{
		Name: "handler-service",
		Port: 8080,
		RoutesConfig: &types.RoutesConfig{
			RoutesGroup: []*types.RoutesGroup{
				{
					Routes: []*types.Route{
						{Method: "POST", Path: "/login", Handler: "Login"},
						{Method: "GET", Path: "/users", Handler: "GetUsers"},
						{Method: "POST", Path: "/logout", Handler: "Logout"},
					},
				},
			},
		},
	}

	pkg := DefaultHandlersPackage(svc)

	if pkg == nil {
		t.Fatal("Handlers package should not be nil")
	}

	if pkg.Name != "handlers" {
		t.Errorf("Package name = %q, want %q", pkg.Name, "handlers")
	}

	if len(pkg.Files) != 3 {
		t.Errorf("Should have 3 handler files, got %d", len(pkg.Files))
	}

	// Verify file names
	fileNames := make(map[string]bool)
	for _, f := range pkg.Files {
		fileNames[f.Name] = true
	}

	expectedFiles := []string{"Login.go", "GetUsers.go", "Logout.go"}
	for _, expected := range expectedFiles {
		if !fileNames[expected] {
			t.Errorf("Missing handler file: %s", expected)
		}
	}

	// Verify handler content
	for _, f := range pkg.Files {
		rendered := mustRenderAST(t, f.Content)
		mustValidateGoCode(t, rendered)
		if !strings.Contains(rendered, "package handlers") {
			t.Errorf("Handler file %s should have package handlers", f.Name)
		}
		if !strings.Contains(rendered, "http.ResponseWriter") {
			t.Errorf("Handler file %s should have http.ResponseWriter parameter", f.Name)
		}
		if !strings.Contains(rendered, "*http.Request") {
			t.Errorf("Handler file %s should have *http.Request parameter", f.Name)
		}
	}
}

// TestDefaultHandlersPackageDeduplication tests that duplicate handlers are skipped
func TestDefaultHandlersPackageDeduplication(t *testing.T) {
	svc := &types.Service{
		Name: "dedup-service",
		Port: 8080,
		RoutesConfig: &types.RoutesConfig{
			RoutesGroup: []*types.RoutesGroup{
				{
					Routes: []*types.Route{
						{Method: "POST", Path: "/login", Handler: "Auth"},
						{Method: "POST", Path: "/refresh", Handler: "Auth"}, // Duplicate handler
						{Method: "GET", Path: "/logout", Handler: "Auth"},   // Duplicate handler
					},
				},
			},
		},
	}

	pkg := DefaultHandlersPackage(svc)

	if pkg == nil {
		t.Fatal("Handlers package should not be nil")
	}

	// Should only have 1 file (Auth.go) since all handlers are the same
	if len(pkg.Files) != 1 {
		t.Errorf("Should have 1 handler file (deduplicated), got %d", len(pkg.Files))
	}

	if pkg.Files[0].Name != "Auth.go" {
		t.Errorf("File name = %q, want %q", pkg.Files[0].Name, "Auth.go")
	}
}

// TestDefaultHandlersPackageWithoutRoutes tests handlers package without routes
func TestDefaultHandlersPackageWithoutRoutes(t *testing.T) {
	svc := &types.Service{
		Name: "no-routes-service",
		Port: 8080,
	}

	pkg := DefaultHandlersPackage(svc)

	if pkg != nil {
		t.Error("Handlers package should be nil when no routes configured")
	}
}

// TestDefaultHandlersPackageEmptyHandlers tests handlers package with empty handler names
func TestDefaultHandlersPackageEmptyHandlers(t *testing.T) {
	svc := &types.Service{
		Name: "empty-handlers-service",
		Port: 8080,
		RoutesConfig: &types.RoutesConfig{
			RoutesGroup: []*types.RoutesGroup{
				{
					Routes: []*types.Route{
						{Method: "POST", Path: "/login", Handler: ""}, // Empty handler
					},
				},
			},
		},
	}

	pkg := DefaultHandlersPackage(svc)

	if pkg != nil {
		t.Error("Handlers package should be nil when all handlers are empty")
	}
}

// TestWithPostgres tests WithPostgres option
func TestWithPostgres(t *testing.T) {
	svc := &types.Service{Name: "pg-service"}
	WithPostgres()(svc)

	if svc.DB == nil {
		t.Fatal("DB should be set")
	}
	if svc.DB.Driver != "pgx" {
		t.Errorf("DB.Driver = %q, want %q", svc.DB.Driver, "pgx")
	}
	if svc.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", svc.DB.Port, 5432)
	}

	// Should add config package
	hasConfig := false
	for _, pkg := range svc.Packages {
		if pkg.Name == "config" {
			hasConfig = true
			break
		}
	}
	if !hasConfig {
		t.Error("WithPostgres should add config package")
	}
}

// TestWithMain tests WithMain option
func TestWithMain(t *testing.T) {
	svc := &types.Service{Name: "main-service"}
	WithMain()(svc)

	hasCmd := false
	for _, pkg := range svc.Packages {
		if pkg.Name == "cmd" {
			hasCmd = true
			// Verify it has main.go
			hasMain := false
			for _, f := range pkg.Files {
				if f.Name == "main.go" {
					hasMain = true
					break
				}
			}
			if !hasMain {
				t.Error("cmd package should have main.go file")
			}
			break
		}
	}
	if !hasCmd {
		t.Error("WithMain should add cmd package")
	}
}

// TestWithRoutes tests WithRoutes option
func TestWithRoutes(t *testing.T) {
	svc := &types.Service{Name: "routes-service"}
	WithRoutes()(svc)

	hasRoutes := false
	for _, pkg := range svc.Packages {
		if pkg.Name == "routes" {
			hasRoutes = true
			// Verify it has routes.go
			hasRoutesFile := false
			for _, f := range pkg.Files {
				if f.Name == "routes.go" {
					hasRoutesFile = true
					break
				}
			}
			if !hasRoutesFile {
				t.Error("routes package should have routes.go file")
			}
			break
		}
	}
	if !hasRoutes {
		t.Error("WithRoutes should add routes package")
	}
}

// TestWithHandlers tests WithHandlers option
func TestWithHandlers(t *testing.T) {
	svc := &types.Service{
		Name: "handlers-service",
		RoutesConfig: &types.RoutesConfig{
			RoutesGroup: []*types.RoutesGroup{
				{
					Routes: []*types.Route{
						{Method: "GET", Path: "/test", Handler: "TestHandler"},
					},
				},
			},
		},
	}
	WithHandlers()(svc)

	hasHandlers := false
	for _, pkg := range svc.Packages {
		if pkg != nil && pkg.Name == "handlers" {
			hasHandlers = true
			break
		}
	}
	if !hasHandlers {
		t.Error("WithHandlers should add handlers package")
	}
}

// TestRouteMethodCall tests route method call generation
func TestRouteMethodCall(t *testing.T) {
	tests := []struct {
		method       string
		path         string
		handler      string
		wantNil      bool
		wantContains string
	}{
		{"POST", "/login", "Login", false, "Post"},
		{"GET", "/users", "GetUsers", false, "Get"},
		{"PUT", "/user", "UpdateUser", false, "Put"},
		{"DELETE", "/user", "DeleteUser", false, "Delete"},
		{"PATCH", "/user", "PatchUser", false, "Patch"},
		{"OPTIONS", "/", "Options", false, "Options"},
		{"INVALID", "/test", "Test", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := routeMethodCall("mux", tt.method, tt.path, tt.handler)

			if tt.wantNil {
				if result != nil {
					t.Errorf("routeMethodCall() should return nil for method %s", tt.method)
				}
				return
			}

			if result == nil {
				t.Fatalf("routeMethodCall() returned nil for method %s", tt.method)
			}
		})
	}
}

// TestAvailableTemplates tests the predefined templates
func TestAvailableTemplates(t *testing.T) {
	if len(AvailableTemplates) == 0 {
		t.Fatal("AvailableTemplates should not be empty")
	}

	expectedIDs := []string{"auth", "custom", "broker", "listener"}
	for _, expectedID := range expectedIDs {
		found := false
		for _, tmpl := range AvailableTemplates {
			if tmpl.ID == expectedID {
				found = true
				if tmpl.Service == nil {
					t.Errorf("Template %s should have a Service", expectedID)
				}
				break
			}
		}
		if !found {
			t.Errorf("Missing template with ID: %s", expectedID)
		}
	}
}

// TestTemplatesDynamicNames verifies templates don't have hardcoded wrong names
func TestTemplatesDynamicNames(t *testing.T) {
	for _, tmpl := range AvailableTemplates {
		if tmpl.Service == nil {
			continue
		}

		// Find cmd package and check main.go import
		for _, pkg := range tmpl.Service.Packages {
			if pkg.Name == "cmd" {
				for _, f := range pkg.Files {
					if f.Name == "main.go" && f.Content != nil {
						rendered := mustRenderAST(t, f.Content)
						mustValidateGoCode(t, rendered)
						// Should import the correct service config, not a hardcoded one
						expectedImport := tmpl.Service.Name + "/config"
						if !strings.Contains(rendered, expectedImport) {
							t.Errorf("Template %s: main.go should import %q", tmpl.ID, expectedImport)
						}
					}
				}
			}
		}
	}
}

// TestBrokerService tests the BrokerService constructor
func TestBrokerService(t *testing.T) {
	svc := BrokerService(
		WithName("my-broker"),
		WithPort(8082),
	)

	if svc.Name != "my-broker" {
		t.Errorf("Name = %q, want %q", svc.Name, "my-broker")
	}
	if svc.Port != 8082 {
		t.Errorf("Port = %d, want %d", svc.Port, 8082)
	}

	// Broker should NOT have a DB
	if svc.DB != nil {
		t.Error("Broker service should not have DB configured")
	}

	// Should have config, cmd, routes, event packages
	packageNames := make(map[string]bool)
	for _, pkg := range svc.Packages {
		packageNames[pkg.Name] = true
	}

	expectedPackages := []string{"config", "cmd", "routes", "event"}
	for _, expected := range expectedPackages {
		if !packageNames[expected] {
			t.Errorf("Broker service missing package: %s", expected)
		}
	}
}

// TestBrokerEventPackage tests the broker event package generation
func TestBrokerEventPackage(t *testing.T) {
	svc := &types.Service{Name: "broker-service"}
	pkg := BrokerEventPackage(svc)

	if pkg.Name != "event" {
		t.Errorf("Package name = %q, want %q", pkg.Name, "event")
	}

	if len(pkg.Files) != 2 {
		t.Errorf("Should have 2 files (emitter.go, event.go), got %d", len(pkg.Files))
	}

	fileNames := make(map[string]bool)
	for _, f := range pkg.Files {
		fileNames[f.Name] = true
	}

	if !fileNames["emitter.go"] {
		t.Error("Missing emitter.go")
	}
	if !fileNames["event.go"] {
		t.Error("Missing event.go")
	}
}

// TestBrokerEmitterFile tests the emitter.go generation for broker
func TestBrokerEmitterFile(t *testing.T) {
	svc := &types.Service{Name: "broker-service"}
	pkg := BrokerEventPackage(svc)

	var emitterFile *types.File
	for _, f := range pkg.Files {
		if f.Name == "emitter.go" {
			emitterFile = f
			break
		}
	}

	if emitterFile == nil {
		t.Fatal("emitter.go not found")
	}

	rendered := mustRenderAST(t, emitterFile.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package event",
		"type Emitter struct",
		"func NewEmitter",
		"func (e *Emitter) Push",
		"func (e *Emitter) SendResponse",
		"github.com/rabbitmq/amqp091-go",
		"github.com/google/uuid",
		"amqp.Connection",
		"amqp.Publishing",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("emitter.go should contain %q", part)
		}
	}
}

// TestBrokerEventFile tests the event.go generation for broker
func TestBrokerEventFile(t *testing.T) {
	svc := &types.Service{Name: "broker-service"}
	pkg := BrokerEventPackage(svc)

	var eventFile *types.File
	for _, f := range pkg.Files {
		if f.Name == "event.go" {
			eventFile = f
			break
		}
	}

	if eventFile == nil {
		t.Fatal("event.go not found")
	}

	rendered := mustRenderAST(t, eventFile.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package event",
		"type EventPayload struct",
		"type TopicPayload struct",
		"func ConnectToRabbit",
		"func SendToListener",
		"json.RawMessage",
		"amqp.Dial",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("event.go should contain %q", part)
		}
	}
}

// TestBrokerConfigFile tests the config.go generation for broker
func TestBrokerConfigFile(t *testing.T) {
	svc := &types.Service{Name: "broker-service"}
	file := BrokerConfigFile(svc)

	if file.Name != "config.go" {
		t.Errorf("File name = %q, want %q", file.Name, "config.go")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package config",
		"broker-service/routes",
		"type Config struct",
		"func InitConfig",
		"func (app *Config) InitServer",
		"routes.Routes",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Broker config.go should contain %q, got:\n%s", part, rendered)
		}
	}

	// Should NOT have database-related code
	if strings.Contains(rendered, "database/sql") {
		t.Error("Broker config.go should not have database imports")
	}
	if strings.Contains(rendered, "connectToDB") {
		t.Error("Broker config.go should not have connectToDB")
	}
}

// TestBrokerMainFile tests the main.go generation for broker
func TestBrokerMainFile(t *testing.T) {
	svc := &types.Service{Name: "broker-service"}
	file := BrokerMainFile(svc)

	if file.Name != "main.go" {
		t.Errorf("File name = %q, want %q", file.Name, "main.go")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package main",
		"broker-service/config",
		"func main()",
		"config.InitConfig",
		"InitServer",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Broker main.go should contain %q", part)
		}
	}
}

// TestListenerService tests the ListenerService constructor
func TestListenerService(t *testing.T) {
	svc := ListenerService(
		WithName("my-listener"),
	)

	if svc.Name != "my-listener" {
		t.Errorf("Name = %q, want %q", svc.Name, "my-listener")
	}

	// Listener should NOT have a DB
	if svc.DB != nil {
		t.Error("Listener service should not have DB configured")
	}

	// Should have config, cmd, event packages (no routes for listener)
	packageNames := make(map[string]bool)
	for _, pkg := range svc.Packages {
		packageNames[pkg.Name] = true
	}

	expectedPackages := []string{"config", "cmd", "event"}
	for _, expected := range expectedPackages {
		if !packageNames[expected] {
			t.Errorf("Listener service missing package: %s", expected)
		}
	}

	// Should NOT have routes package
	if packageNames["routes"] {
		t.Error("Listener service should not have routes package")
	}
}

// TestListenerEventPackage tests the listener event package generation
func TestListenerEventPackage(t *testing.T) {
	svc := &types.Service{Name: "listener-service"}
	pkg := ListenerEventPackage(svc)

	if pkg.Name != "event" {
		t.Errorf("Package name = %q, want %q", pkg.Name, "event")
	}

	if len(pkg.Files) != 2 {
		t.Errorf("Should have 2 files (consumer.go, event.go), got %d", len(pkg.Files))
	}

	fileNames := make(map[string]bool)
	for _, f := range pkg.Files {
		fileNames[f.Name] = true
	}

	if !fileNames["consumer.go"] {
		t.Error("Missing consumer.go")
	}
	if !fileNames["event.go"] {
		t.Error("Missing event.go")
	}
}

// TestListenerConsumerFile tests the consumer.go generation for listener
func TestListenerConsumerFile(t *testing.T) {
	svc := &types.Service{Name: "listener-service"}
	pkg := ListenerEventPackage(svc)

	var consumerFile *types.File
	for _, f := range pkg.Files {
		if f.Name == "consumer.go" {
			consumerFile = f
			break
		}
	}

	if consumerFile == nil {
		t.Fatal("consumer.go not found")
	}

	rendered := mustRenderAST(t, consumerFile.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package event",
		"type handler map",
		"type Consumer struct",
		"func NewConsumer",
		"func (c *Consumer) Setup",
		"func (c *Consumer) Listen",
		"github.com/rabbitmq/amqp091-go",
		"amqp.Connection",
		"amqp.Delivery",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("consumer.go should contain %q", part)
		}
	}
}

// TestListenerEventFile tests the event.go generation for listener
func TestListenerEventFile(t *testing.T) {
	svc := &types.Service{Name: "listener-service"}
	pkg := ListenerEventPackage(svc)

	var eventFile *types.File
	for _, f := range pkg.Files {
		if f.Name == "event.go" {
			eventFile = f
			break
		}
	}

	if eventFile == nil {
		t.Fatal("event.go not found")
	}

	rendered := mustRenderAST(t, eventFile.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package event",
		"type EventPayload struct",
		"func ConnectToRabbit",
		"func (c *Consumer) handlePayload",
		"json.RawMessage",
		"amqp.Dial",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("listener event.go should contain %q", part)
		}
	}
}

// TestListenerConfigFile tests the config.go generation for listener
func TestListenerConfigFile(t *testing.T) {
	svc := &types.Service{Name: "listener-service"}
	file := ListenerConfigFile(svc)

	if file.Name != "config.go" {
		t.Errorf("File name = %q, want %q", file.Name, "config.go")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package config",
		"listener-service/event",
		"type Config struct",
		"Conn",
		"amqp.Connection",
		"func InitConfig",
		"func (app *Config) StartListening",
		"event.ConnectToRabbit",
		"event.NewConsumer",
		"consumer.Setup",
		"consumer.Listen",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Listener config.go should contain %q, got:\n%s", part, rendered)
		}
	}

	// Should NOT have HTTP server code
	if strings.Contains(rendered, "InitServer") {
		t.Error("Listener config.go should not have InitServer")
	}
	if strings.Contains(rendered, "http.Server") {
		t.Error("Listener config.go should not have http.Server")
	}
}

// TestListenerMainFile tests the main.go generation for listener
func TestListenerMainFile(t *testing.T) {
	svc := &types.Service{Name: "listener-service"}
	file := ListenerMainFile(svc)

	if file.Name != "main.go" {
		t.Errorf("File name = %q, want %q", file.Name, "main.go")
	}

	rendered := mustRenderAST(t, file.Content)
	mustValidateGoCode(t, rendered)

	expectedParts := []string{
		"package main",
		"listener-service/config",
		"func main()",
		"config.InitConfig",
		"StartListening",
	}

	for _, part := range expectedParts {
		if !strings.Contains(rendered, part) {
			t.Errorf("Listener main.go should contain %q", part)
		}
	}

	// Should NOT call InitServer
	if strings.Contains(rendered, "InitServer") {
		t.Error("Listener main.go should not call InitServer")
	}
}

// TestWithBrokerEvent tests the WithBrokerEvent option
func TestWithBrokerEvent(t *testing.T) {
	svc := &types.Service{Name: "test-broker"}
	WithBrokerEvent()(svc)

	hasEvent := false
	for _, pkg := range svc.Packages {
		if pkg.Name == "event" {
			hasEvent = true
			// Verify it has emitter.go and event.go
			fileNames := make(map[string]bool)
			for _, f := range pkg.Files {
				fileNames[f.Name] = true
			}
			if !fileNames["emitter.go"] {
				t.Error("Broker event package should have emitter.go")
			}
			if !fileNames["event.go"] {
				t.Error("Broker event package should have event.go")
			}
			break
		}
	}
	if !hasEvent {
		t.Error("WithBrokerEvent should add event package")
	}
}

// TestWithListenerEvent tests the WithListenerEvent option
func TestWithListenerEvent(t *testing.T) {
	svc := &types.Service{Name: "test-listener"}
	WithListenerEvent()(svc)

	hasEvent := false
	for _, pkg := range svc.Packages {
		if pkg.Name == "event" {
			hasEvent = true
			// Verify it has consumer.go and event.go
			fileNames := make(map[string]bool)
			for _, f := range pkg.Files {
				fileNames[f.Name] = true
			}
			if !fileNames["consumer.go"] {
				t.Error("Listener event package should have consumer.go")
			}
			if !fileNames["event.go"] {
				t.Error("Listener event package should have event.go")
			}
			break
		}
	}
	if !hasEvent {
		t.Error("WithListenerEvent should add event package")
	}
}

// TestBrokerTemplateStructure tests the broker template from AvailableTemplates
func TestBrokerTemplateStructure(t *testing.T) {
	var brokerTemplate *Template
	for _, tmpl := range AvailableTemplates {
		if tmpl.ID == "broker" {
			brokerTemplate = tmpl
			break
		}
	}

	if brokerTemplate == nil {
		t.Fatal("Broker template not found")
	}

	svc := brokerTemplate.Service

	// Check expected packages exist
	packageNames := make(map[string]bool)
	for _, pkg := range svc.Packages {
		packageNames[pkg.Name] = true
	}

	if !packageNames["event"] {
		t.Error("Broker template should have event package")
	}
	if !packageNames["routes"] {
		t.Error("Broker template should have routes package")
	}
	if !packageNames["config"] {
		t.Error("Broker template should have config package")
	}
	if !packageNames["cmd"] {
		t.Error("Broker template should have cmd package")
	}
}

// TestListenerTemplateStructure tests the listener template from AvailableTemplates
func TestListenerTemplateStructure(t *testing.T) {
	var listenerTemplate *Template
	for _, tmpl := range AvailableTemplates {
		if tmpl.ID == "listener" {
			listenerTemplate = tmpl
			break
		}
	}

	if listenerTemplate == nil {
		t.Fatal("Listener template not found")
	}

	svc := listenerTemplate.Service

	// Check expected packages exist
	packageNames := make(map[string]bool)
	for _, pkg := range svc.Packages {
		packageNames[pkg.Name] = true
	}

	if !packageNames["event"] {
		t.Error("Listener template should have event package")
	}
	if !packageNames["config"] {
		t.Error("Listener template should have config package")
	}
	if !packageNames["cmd"] {
		t.Error("Listener template should have cmd package")
	}

	// Listener should NOT have routes
	if packageNames["routes"] {
		t.Error("Listener template should NOT have routes package")
	}
}

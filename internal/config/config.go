package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/flaviogonzalez/instant-layer/internal/types"
)

// top config
type Layer struct {
	Name        string    `json:"name"` // project name
	Root        string    `json:"root"`
	GeneratedAt time.Time `json:"generated_at"`
	Services    []*types.Service
}

func (l *Layer) Save() error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	// Create directory only if it doesn't exist
	if _, err := os.Stat(l.Root); os.IsNotExist(err) {
		if err := os.MkdirAll(l.Root, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(filepath.Join(l.Root, "layer.json"), data, 0644)
}

// Update saves the layer.json without creating the root directory
func (l *Layer) Update() error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(l.Root, "layer.json"), data, 0644)
}

func (l *Layer) RegenerateDockerCompose() error {
	const tmpl = `name: {{.Name}}

services:
{{range .Services}}  {{.Name}}:
    build: ./{{.Dir}}
    ports:
      - "{{.Port}}:{{.Port}}"
    restart: unless-stopped
{{end}}
networks:
  default:
    name: {{.Name}}-network
`

	t := template.Must(template.New("dc").Parse(tmpl))
	path := filepath.Join(l.Root, "docker-compose.yml")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, l)
}

func (l *Layer) Reload() error {
	path := filepath.Join(l.Root, "layer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, l)
}

func FindLayerRoot(start string) (string, error) {
	dir := start
	for {
		path := filepath.Join(dir, "layer.json")
		if _, err := os.Stat(path); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no layer.json found in any parent directory")
		}
		dir = parent
	}
}

func (l *Layer) Hydrate() error {
	services, err := l.ScanServices()
	if err != nil {
		return err
	}
	l.Services = services

	// Scan routes and benchmarks for each service
	for _, svc := range l.Services {
		servicePath := filepath.Join(l.Root, svc.Name)

		// Parse routes from routes.go files
		routesConfig := l.scanRoutes(servicePath)
		if routesConfig != nil {
			svc.RoutesConfig = routesConfig
		}

		// Analyze benchmark metrics
		benchmark := l.analyzeServiceBenchmark(servicePath, svc)
		svc.Benchmark = benchmark
	}

	return nil
}

func (l *Layer) ScanServices() ([]*types.Service, error) {
	entries, err := os.ReadDir(l.Root)
	if err != nil {
		return nil, err
	}

	var services []*types.Service

	for _, entry := range entries {
		// Skip files and hidden directories
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		servicePath := filepath.Join(l.Root, entry.Name())

		// Check if this is a valid service directory (has go.mod)
		service, err := l.parseServiceDir(servicePath, entry.Name())
		if err != nil {
			continue // Skip directories that aren't valid services
		}

		if service != nil {
			services = append(services, service)
		}
	}

	return services, nil
}

// parseServiceDir attempts to parse a directory as a service
func (l *Layer) parseServiceDir(servicePath, dirName string) (*types.Service, error) {
	modPath := filepath.Join(servicePath, "go.mod")

	// Check if go.mod exists
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no go.mod found")
	}

	modData, err := os.ReadFile(modPath)
	if err != nil {
		return nil, err
	}

	moduleName := parseModuleName(string(modData))
	if moduleName == "" {
		return nil, fmt.Errorf("invalid go.mod format")
	}

	service := &types.Service{
		Name: dirName,
	}

	// Scan packages within the service
	service.Packages = l.scanPackages(servicePath)

	return service, nil
}

// parseModuleName extracts module name from go.mod content
func parseModuleName(modContent string) string {
	lines := strings.Split(modContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// scanPackages scans for Go packages within a service directory
func (l *Layer) scanPackages(servicePath string) []*types.Package {
	var packages []*types.Package

	entries, err := os.ReadDir(servicePath)
	if err != nil {
		return packages
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		pkgPath := filepath.Join(servicePath, entry.Name())
		files := l.scanGoFiles(pkgPath)

		if len(files) > 0 {
			packages = append(packages, &types.Package{
				Name:  entry.Name(),
				Files: files,
			})
		}
	}

	return packages
}

// scanGoFiles scans for .go files in a package directory
func (l *Layer) scanGoFiles(pkgPath string) []*types.File {
	var files []*types.File

	entries, err := os.ReadDir(pkgPath)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		files = append(files, &types.File{
			Name:    entry.Name(),
			Content: nil,
		})
	}

	return files
}

// Route patterns for scanning routes.go files
var (
	// Matches mux.Post("/path", handlers.Handler) or mux.Get("/path", handlers.Handler)
	routePattern = regexp.MustCompile(`\.(Post|Get|Put|Delete|Patch|Options|Head)\s*\(\s*"([^"]+)"`)
	// Matches handlers.HandlerName
	handlerPattern = regexp.MustCompile(`handlers\.(\w+)`)
)

// scanRoutes scans the routes package for route definitions
func (l *Layer) scanRoutes(servicePath string) *types.RoutesConfig {
	routesPath := filepath.Join(servicePath, "routes")

	// Check if routes directory exists
	if _, err := os.Stat(routesPath); os.IsNotExist(err) {
		return nil
	}

	var routes []*types.Route

	// Scan all .go files in routes directory
	err := filepath.Walk(routesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		fileRoutes := l.parseRoutesFile(path)
		routes = append(routes, fileRoutes...)
		return nil
	})

	if err != nil || len(routes) == 0 {
		return nil
	}

	return &types.RoutesConfig{
		RoutesGroup: []*types.RoutesGroup{
			{
				Routes: routes,
			},
		},
	}
}

// parseRoutesFile parses a single routes file for route definitions
func (l *Layer) parseRoutesFile(filePath string) []*types.Route {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var routes []*types.Route
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Find route method and path
		routeMatches := routePattern.FindStringSubmatch(line)
		if len(routeMatches) < 3 {
			continue
		}

		method := strings.ToUpper(routeMatches[1])
		path := routeMatches[2]

		// Find handler name
		handlerName := ""
		handlerMatches := handlerPattern.FindStringSubmatch(line)
		if len(handlerMatches) >= 2 {
			handlerName = handlerMatches[1]
		}

		routes = append(routes, &types.Route{
			Method:  method,
			Path:    path,
			Handler: handlerName,
		})
	}

	return routes
}

// analyzeServiceBenchmark analyzes metrics for a service
func (l *Layer) analyzeServiceBenchmark(servicePath string, svc *types.Service) *types.Benchmark {
	benchmark := &types.Benchmark{
		ServiceName:  svc.Name,
		LastAnalyzed: time.Now(),
	}

	// Count packages
	benchmark.TotalPackages = len(svc.Packages)

	// Walk through all files
	err := filepath.Walk(servicePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".go") {
			benchmark.TotalFiles++

			// Check if it's a test file
			if strings.HasSuffix(path, "_test.go") {
				benchmark.HasTests = true
				benchmark.TestFiles++
			}

			// Count lines
			lines, _ := countLines(path)
			benchmark.TotalLines += lines
		}

		return nil
	})

	if err != nil {
		return benchmark
	}

	// Count dependencies from go.mod
	benchmark.Dependencies = l.countDependencies(servicePath)

	// Try to get binary size if exists
	binaryPath := filepath.Join(servicePath, svc.Name)
	if info, err := os.Stat(binaryPath); err == nil {
		benchmark.BinarySize = info.Size()
	}
	// Also check for .exe on Windows
	binaryPathExe := filepath.Join(servicePath, svc.Name+".exe")
	if info, err := os.Stat(binaryPathExe); err == nil {
		benchmark.BinarySize = info.Size()
	}

	return benchmark
}

// countLines counts the number of lines in a file
func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// countDependencies counts the number of dependencies in go.mod
func (l *Layer) countDependencies(servicePath string) int {
	modPath := filepath.Join(servicePath, "go.mod")
	content, err := os.ReadFile(modPath)
	if err != nil {
		return 0
	}

	count := 0
	lines := strings.Split(string(content), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			count++
		}
		// Single line require
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			count++
		}
	}

	return count
}

// connectionPattern matches URLs like http://service-name/route or grpc://service-name
var connectionPattern = regexp.MustCompile(`(?i)(https?|grpc|wss?|rpc)://([a-z0-9][-a-z0-9]*)(:[0-9]+)?(/[a-zA-Z0-9/_-]*)?`)

// ScanConnections scans all services for inter-service connections
func (l *Layer) ScanConnections() ([]*types.Connection, error) {
	var connections []*types.Connection

	// Build a map of known services for validation
	serviceMap := make(map[string]*types.Service)
	for _, svc := range l.Services {
		serviceMap[svc.Name] = svc
	}

	// Scan each service's source files
	for _, svc := range l.Services {
		servicePath := filepath.Join(l.Root, svc.Name)
		conns, err := l.scanServiceConnections(servicePath, svc.Name, serviceMap)
		if err != nil {
			continue
		}
		connections = append(connections, conns...)
	}

	return connections, nil
}

// scanServiceConnections scans a single service directory for connections
func (l *Layer) scanServiceConnections(servicePath, serviceName string, serviceMap map[string]*types.Service) ([]*types.Connection, error) {
	var connections []*types.Connection

	err := filepath.Walk(servicePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		fileConns, err := l.scanFileConnections(path, serviceName, serviceMap)
		if err != nil {
			return nil // Continue scanning other files
		}
		connections = append(connections, fileConns...)
		return nil
	})

	return connections, err
}

// scanFileConnections scans a single Go file for connection patterns
func (l *Layer) scanFileConnections(filePath, fromService string, serviceMap map[string]*types.Service) ([]*types.Connection, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var connections []*types.Connection
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := connectionPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			protocol := strings.ToLower(match[1])
			targetService := match[2]
			route := ""
			if len(match) >= 5 {
				route = match[4]
			}

			// Skip if connecting to self or not a known service pattern
			if targetService == fromService {
				continue
			}

			conn := &types.Connection{
				FromService: fromService,
				ToService:   targetService,
				Protocol:    parseProtocol(protocol),
				Route:       route,
				SourceFile:  filePath,
				SourceLine:  lineNum,
			}

			// Validate the connection
			l.validateConnection(conn, serviceMap)
			connections = append(connections, conn)
		}
	}

	return connections, scanner.Err()
}

// parseProtocol converts string to ConnectionProtocol
func parseProtocol(p string) types.ConnectionProtocol {
	switch p {
	case "http":
		return types.ProtocolHTTP
	case "https":
		return types.ProtocolHTTPS
	case "grpc":
		return types.ProtocolGRPC
	case "ws", "wss":
		return types.ProtocolWebSocket
	case "rpc":
		return types.ProtocolRPC
	default:
		return types.ProtocolHTTP
	}
}

// validateConnection checks if target service and route exist
func (l *Layer) validateConnection(conn *types.Connection, serviceMap map[string]*types.Service) {
	targetSvc, exists := serviceMap[conn.ToService]
	if !exists {
		conn.Valid = false
		conn.Error = fmt.Sprintf("target service '%s' does not exist", conn.ToService)
		return
	}

	// If no route specified, connection is valid (just connecting to service)
	if conn.Route == "" || conn.Route == "/" {
		conn.Valid = true
		return
	}

	// Check if route exists in target service
	if targetSvc.RoutesConfig != nil {
		for _, group := range targetSvc.RoutesConfig.RoutesGroup {
			for _, route := range group.Routes {
				fullPath := group.Prefix + route.Path
				if fullPath == conn.Route || route.Path == conn.Route {
					conn.Valid = true
					conn.Method = route.Method
					return
				}
			}
		}
	}

	conn.Valid = false
	conn.Error = fmt.Sprintf("route '%s' not found in service '%s'", conn.Route, conn.ToService)
}

// GetServiceDependencies returns services that a given service depends on
func (l *Layer) GetServiceDependencies(serviceName string) []string {
	connections, _ := l.ScanConnections()

	deps := make(map[string]bool)
	for _, conn := range connections {
		if conn.FromService == serviceName && conn.Valid {
			deps[conn.ToService] = true
		}
	}

	var result []string
	for dep := range deps {
		result = append(result, dep)
	}
	return result
}

// ValidateAllConnections returns all invalid connections in the project
func (l *Layer) ValidateAllConnections() ([]*types.Connection, error) {
	connections, err := l.ScanConnections()
	if err != nil {
		return nil, err
	}

	var invalid []*types.Connection
	for _, conn := range connections {
		if !conn.Valid {
			invalid = append(invalid, conn)
		}
	}
	return invalid, nil
}

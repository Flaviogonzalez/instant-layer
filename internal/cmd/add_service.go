package cmd

import (
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/flaviogonzalez/instant-layer/internal/config"
	defaults "github.com/flaviogonzalez/instant-layer/internal/default"
	"github.com/flaviogonzalez/instant-layer/internal/templ"
	"github.com/flaviogonzalez/instant-layer/internal/types"
	"github.com/manifoldco/promptui"
)

func SelectAndGenerateTemplate(root string) error {
	// First, find the layer root (where layer.json is)
	layerRoot, err := config.FindLayerRoot(root)
	if err != nil {
		return fmt.Errorf("layer.json not found. Run 'layer new' first to create a project")
	}

	if len(defaults.AvailableTemplates) == 0 {
		return fmt.Errorf("no templates available")
	}

	selected, err := selectTemplate(
		"Select the type of service you want to create",
		defaults.AvailableTemplates,
	)
	if err != nil {
		return err
	}

	serviceName, err := promptServiceName(selected.ID)
	if err != nil {
		return err
	}

	servicePort, err := promptServicePort(selected.Service.Port)
	if err != nil {
		return err
	}

	var service *types.Service
	switch selected.ID {
	case "broker":
		service = defaults.BrokerService(
			defaults.WithName(serviceName),
			defaults.WithPort(servicePort),
		)
	case "listener":
		service = defaults.ListenerService(
			defaults.WithName(serviceName),
		)
	default:
		service = defaults.DefaultService(
			defaults.WithName(serviceName),
			defaults.WithPort(servicePort),
		)
	}

	servicePath := filepath.Join(layerRoot, serviceName)
	if err := os.MkdirAll(servicePath, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Write the service files (packages)
	if err := WriteService(servicePath, service); err != nil {
		return err
	}

	// Generate go.mod for the service
	if err := generateServiceGoMod(servicePath, serviceName, selected.ID); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	// Update layer.json with the new service
	if err := updateLayerWithService(layerRoot, service); err != nil {
		return fmt.Errorf("failed to update layer.json: %w", err)
	}

	// Regenerate docker-compose.yml
	if err := regenerateDockerCompose(layerRoot); err != nil {
		return fmt.Errorf("failed to regenerate docker-compose: %w", err)
	}

	fmt.Printf("Service '%s' created successfully at port %d!\n", serviceName, servicePort)
	return nil
}

// generateServiceGoMod creates a go.mod file for the service
func generateServiceGoMod(servicePath, serviceName, templateID string) error {
	goModPath := filepath.Join(servicePath, "go.mod")

	// Get default dependencies based on what the service uses
	deps := templ.DefaultDependencies()

	// Add dependencies based on template type
	switch templateID {
	case "broker", "listener":
		// RabbitMQ dependency
		deps = append(deps, templ.Dependency{
			Path:    "github.com/rabbitmq/amqp091-go",
			Version: "v1.10.0",
		})
		if templateID == "broker" {
			// UUID for correlation IDs
			deps = append(deps, templ.Dependency{
				Path:    "github.com/google/uuid",
				Version: "v1.6.0",
			})
		}
	default:
		// Default services use postgres
		deps = append(deps, templ.Dependency{
			Path:    "github.com/jackc/pgx/v5",
			Version: "v5.6.0",
		})
	}

	data := templ.GoModData{
		Name:         serviceName,
		GoVersion:    templ.DefaultGoVersion(),
		Dependencies: deps,
	}

	return templ.GenerateGoMod(goModPath, data)
}

// updateLayerWithService adds a service to layer.json
func updateLayerWithService(root string, service *types.Service) error {
	layer := &config.Layer{Root: root}

	// Try to reload existing layer.json
	if err := layer.Reload(); err != nil {
		return fmt.Errorf("failed to reload layer.json: %w", err)
	}

	// Add the new service
	layer.Services = append(layer.Services, service)

	// Save the updated layer
	return layer.Update()
}

// regenerateDockerCompose regenerates docker-compose.yml from layer.json
func regenerateDockerCompose(root string) error {
	layer := &config.Layer{Root: root}

	// Reload to get latest services
	if err := layer.Reload(); err != nil {
		return err
	}

	// If no services, use Hydrate to scan for services
	if len(layer.Services) == 0 {
		if err := layer.Hydrate(); err != nil {
			return err
		}
	}

	// Build docker-compose data
	var serviceData []*templ.ServiceData
	for _, svc := range layer.Services {
		sd := &templ.ServiceData{
			Name: svc.Name,
			Port: svc.Port,
			DB:   svc.DB,
		}

		// Get dependencies for this service
		deps := layer.GetServiceDependencies(svc.Name)
		sd.DependsOn = deps

		serviceData = append(serviceData, sd)
	}

	data := templ.DockerComposeData{
		Name:     layer.Name,
		Services: serviceData,
	}

	dockerComposePath := filepath.Join(root, "docker-compose.yml")
	return templ.GenerateDockerCompose(dockerComposePath, data)
}

// promptServicePort prompts for service port
func promptServicePort(defaultPort int) (int, error) {
	prompt := promptui.Prompt{
		Label:   fmt.Sprintf("Service port (default: %d)", defaultPort),
		Default: fmt.Sprintf("%d", defaultPort),
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return nil // use default
			}
			var port int
			if _, err := fmt.Sscanf(input, "%d", &port); err != nil {
				return fmt.Errorf("invalid port number")
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("port must be between 1 and 65535")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		return 0, fmt.Errorf("prompt cancelled: %w", err)
	}

	if strings.TrimSpace(result) == "" {
		return defaultPort, nil
	}

	var port int
	fmt.Sscanf(result, "%d", &port)
	return port, nil
}

func WriteService(path string, s *types.Service) error {
	fs := token.NewFileSet()

	for _, p := range s.Packages {
		packagePath := filepath.Join(path, p.Name)

		err := os.Mkdir(packagePath, 0755)
		if err != nil {
			return err
		}

		for _, f := range p.Files {
			file, err := os.Create(filepath.Join(packagePath, f.Name))
			if err != nil {
				return err
			}
			defer file.Close()

			err = format.Node(file, fs, f.Content)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func selectTemplate(label string, templates []*defaults.Template) (*defaults.Template, error) {
	if len(templates) == 0 {
		return &defaults.Template{}, fmt.Errorf("no templates available")
	}

	templatesCfg := &promptui.SelectTemplates{
		Label:    "{{ . | cyan }}",
		Active:   "→ {{ .Name | cyan }}",
		Inactive: "  {{ .Name | white }}",
		Selected: "{{ \"Checkmark\" | green }} {{ .Name | green }} — {{ .Description | faint }}",
		Details: `
--------- Template Details ----------
{{ "Name:"         | faint }} {{ .Name }}
{{ "Description:"  | faint }} {{ .Description }}
{{ "ID:"           | faint }} {{ .ID }}`,
	}

	searcher := func(input string, index int) bool {
		t := templates[index]
		input = strings.ToLower(input)
		return strings.Contains(strings.ToLower(t.Name), input) ||
			strings.Contains(strings.ToLower(t.Description), input) ||
			strings.Contains(strings.ToLower(t.ID), input)
	}

	selector := promptui.Select{
		Label:             label,
		Items:             templates,
		Templates:         templatesCfg,
		Size:              10,
		Searcher:          searcher,
		StartInSearchMode: len(templates) > 10, // activamos búsqueda si hay muchos items
	}

	idx, _, err := selector.Run()
	if err != nil {
		var zero defaults.Template
		return &zero, fmt.Errorf("selection cancelled: %w", err)
	}

	return templates[idx], nil
}

func promptServiceName(templateID string) (string, error) {
	defaultName := fmt.Sprintf("%s-service", templateID)

	prompt := promptui.Prompt{
		Label:   fmt.Sprintf("Service name (default: %s)", defaultName),
		Default: defaultName,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("service name cannot be empty")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt cancelled: %w", err)
	}

	if strings.TrimSpace(result) == "" {
		return defaultName, nil
	}

	return result, nil
}

func promptString(label, defaultValue string, validate func(string) error) (string, error) {
	p := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
		Validate: func(input string) error {
			if validate == nil {
				if input == "" && defaultValue == "" {
					return fmt.Errorf("el valor no puede estar vacío")
				}
				return nil
			}
			return validate(input)
		},
	}

	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("entrada cancelada: %w", err)
	}

	if result == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return result, nil
}

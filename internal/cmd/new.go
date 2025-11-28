package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/flaviogonzalez/instant-layer/internal/config"
	"github.com/flaviogonzalez/instant-layer/internal/templ"
	"github.com/flaviogonzalez/instant-layer/internal/types"
	"github.com/manifoldco/promptui"
)

func StartGeneration(wd, dir string) error {
	resolvedRoot, err := resolveProjectRoot(wd, dir)
	if err != nil {
		return err
	}

	// Extract project name from resolved root
	projectName := filepath.Base(resolvedRoot)
	if dir == "./" {
		projectName = filepath.Base(wd)
	}

	layer := &config.Layer{
		Name:        projectName,
		Root:        resolvedRoot,
		Services:    []*types.Service{},
		GeneratedAt: time.Now(),
	}

	if err := layer.Save(); err != nil {
		return fmt.Errorf("failed to save layer config: %w", err)
	}

	fmt.Printf("Project '%s' initialized at: %s\n", projectName, resolvedRoot)

	// Offer to create the first service
	if shouldAddFirstService() {
		if err := addFirstService(layer); err != nil {
			fmt.Printf("Warning: First service not created: %v\n", err)
			fmt.Println("You can add services later with: layer add service")
		}
	} else {
		// Generate empty docker-compose.yml
		if err := generateEmptyDockerCompose(layer); err != nil {
			fmt.Printf("Warning: docker-compose.yml not created: %v\n", err)
		}
	}

	fmt.Printf("Project created successfully at: %s\n", resolvedRoot)
	return nil
}

func shouldAddFirstService() bool {
	prompt := promptui.Prompt{
		Label:     "Would you like to add your first service now?",
		IsConfirm: true,
		Default:   "y",
	}
	result, err := prompt.Run()
	return err == nil && (result == "y" || result == "Y" || result == "")
}

func addFirstService(layer *config.Layer) error {
	fmt.Println("Let's create your first service")
	return SelectAndGenerateTemplate(layer.Root)
}

func generateEmptyDockerCompose(layer *config.Layer) error {
	data := templ.DockerComposeData{
		Name:     layer.Name,
		Services: []*templ.ServiceData{},
	}

	dockerComposePath := filepath.Join(layer.Root, "docker-compose.yml")
	return templ.GenerateDockerCompose(dockerComposePath, data)
}

func resolveProjectRoot(wd, dir string) (string, error) {
	switch dir {
	case "", ".":
		prompt := promptui.Prompt{
			Label: "Project name",
			Validate: func(input string) error {
				input = filepath.Clean(input)
				if input == "" || input == "." || input == ".." {
					return fmt.Errorf("project name cannot be empty or '.'/'..'")
				}
				return nil
			},
		}

		result, err := prompt.Run()
		if err != nil {
			return "", fmt.Errorf("prompt cancelled or failed: %w", err)
		}
		return filepath.Join(wd, filepath.Clean(result)), nil
	case "./":
		return wd, nil

	default:
		return filepath.Join(wd, filepath.Clean(dir)), nil
	}
}

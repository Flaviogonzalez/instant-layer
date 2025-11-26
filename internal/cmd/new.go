package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/flaviogonzalez/instant-layer/internal/config"
	service "github.com/flaviogonzalez/instant-layer/internal/services"
	"github.com/manifoldco/promptui"
)

func StartGeneration(wd, dir string) error {
	resolvedRoot, err := resolveProjectRoot(wd, dir)

	if dir == "" {
		dir = "./"
	}

	if err != nil {
		return err
	}

	layer := &config.Layer{
		Name:        dir,
		Root:        resolvedRoot,
		Services:    []*service.Service{},
		GeneratedAt: time.Now(),
	}

	log.Println(layer)

	if err := layer.Save(); err != nil {
		return fmt.Errorf("failed to save layer config: %w", err)
	}

	// 5. Offer to create the first service
	if shouldAddFirstService() {
		if err := addFirstService(layer); err != nil {
			fmt.Printf("Warning: First service not created: %v\n", err)
			fmt.Println("You can add services later with: layer add service")
		} else {
			// Regenerate docker-compose with the new service
			_ = layer.Reload() // refresh from disk
			_ = layer.RegenerateDockerCompose()
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
	return err == nil && (result == "y" || result == "Y")
}

func addFirstService(layer *config.Layer) error {
	fmt.Println("Let's create your first service")
	return SelectAndGenerateTemplate(layer.Root) // reuse your existing logic, now with layer context
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

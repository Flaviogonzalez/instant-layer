package cmd

import (
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"

	defaults "github.com/flaviogonzalez/instant-layer/internal/default"
	service "github.com/flaviogonzalez/instant-layer/internal/services"
	"github.com/flaviogonzalez/instant-layer/internal/utils"
)

func SelectAndGenerateTemplate(root string) error {
	if len(defaults.AvailableTemplates) == 0 {
		return fmt.Errorf("no templates available")
	}

	selected, err := utils.SelectTemplate(
		"Select the type of service you want to create",
		defaults.AvailableTemplates,
	)
	if err != nil {
		return err
	}

	serviceName, err := utils.PromptServiceName(selected.ID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(root, serviceName), 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	if err := WriteService(filepath.Join(root, serviceName), selected.Service); err != nil {
		return err
	}

	fmt.Printf("Service '%s' created successfully!\n", serviceName)
	return nil
}

func WriteService(path string, s *service.Service) error {
	fs := token.NewFileSet()

	for _, p := range s.Packages {
		packagePath := filepath.Join(path, p.Name)

		err := os.Mkdir(packagePath, 0755)
		if err != nil {
			return err
		}

		for _, f := range p.Files {
			file, err := os.Create(f.Name)
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

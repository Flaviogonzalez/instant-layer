package utils

import (
	"fmt"
	"strings"

	defaults "github.com/flaviogonzalez/instant-layer/internal/default"
	"github.com/manifoldco/promptui"
)

func SelectTemplate(label string, templates []*defaults.Template) (*defaults.Template, error) {
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

// PromptServiceName solicita el nombre del servicio con valor por defecto basado en el ID
func PromptServiceName(templateID string) (string, error) {
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

func PromptString(label, defaultValue string, validate func(string) error) (string, error) {
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

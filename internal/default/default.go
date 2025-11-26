package defaults

import (
	service "github.com/flaviogonzalez/instant-layer/internal/services"
)

type Template struct {
	ID          string
	Name        string
	Description string
	Port        int
	Service     *service.Service
}

var AvailableTemplates = []*Template{
	{
		ID:          "auth",
		Name:        "auth-service",
		Description: "preconfigured auth-service with the following routes: /login, /logout, /register (/send if email-service available)",
		Service:     DefaultService(),
	},
	{
		ID:          "custom",
		Name:        "custom api-service",
		Description: "no-config scaffolding service.",
		Service:     DefaultService(),
	},
	{
		ID:          "broker",
		Name:        "broker-service",
		Description: "preconfigured broker-service with no connections.",
		Service:     DefaultService(),
	},
	{
		ID:          "listener",
		Name:        "listener-service",
		Description: "preconfigured listener-service.",
		Service:     DefaultService(),
	},
	{
		ID:          "cli",
		Name:        "cli-service",
		Description: "preconfigured cli environment with cobra and promptui.",
		Service:     DefaultService(),
	},
}

func DefaultService(override ...func(*service.Service)) *service.Service {
	service := &service.Service{
		Port: 8080,
	}

	// add packages
	service.Packages = append(service.Packages, packages...)

	for _, o := range override {
		if o != nil {
			o(service)
		}
	}

	return service
}

package defaults

import (
	"github.com/flaviogonzalez/instant-layer/internal/routes"
	service "github.com/flaviogonzalez/instant-layer/internal/services"
)

var authService = &service.Service{
	Name: "auth-service",
	Port: 8080,
	DB: &service.Database{
		TimeoutConn: 10,
		Driver:      "pgx",
		URL:         "",
		Port:        8080,
	},
	RoutesConfig: &routes.RoutesConfig{
		CORS: &routes.CorsOptions{
			AllowedOrigins:   []string{},
			AllowedMethods:   []string{},
			AllowedHeaders:   []string{},
			AllowCredentials: true,
			MaxAge:           30,
		},
		RoutesGroup: []*routes.RoutesGroup{
			{
				Prefix: "",
				Routes: []*routes.Route{
					{
						Path:    "/register",
						Method:  "POST",
						Handler: "register",
					},
				},
			},
		},
	},
}

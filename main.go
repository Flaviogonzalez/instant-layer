package main

import (
	"instant-layer/files"
)

func main() {
	outputDir := "dist"
	projectName := "instant-layer"

	genCfg := files.NewGenConfig(
		files.WithService(files.Service{
			Name: "auth-service",
			ServerType: &files.API{
				DB: files.Database{
					TimeoutConn: 15,
					Driver:      "pgx",
					URL:         "",
				},
				RoutesConfig: files.RoutesConfig{
					RoutesGroup: []*files.RoutesGroup{
						&files.RoutesGroup{
							Routes: []*files.Route{
								&files.Route{
									Path:    "/createUser",
									Method:  "POST",
									Handler: "CreateUser", // always first letter uppercase
								},
								&files.Route{
									Path:    "/deleteUser",
									Method:  "DELETE",
									Handler: "DeleteUser", // always first letter uppercase
								},
								&files.Route{
									Path:    "/editUser",
									Method:  "PUT",
									Handler: "EditUser", // always first letter uppercase
								},
							},
						},
						&files.RoutesGroup{
							Routes: []*files.Route{
								&files.Route{
									Path:    "/users",
									Method:  "GET",
									Handler: "GetUsers", // always first letter uppercase
								},
							},
						},
					},
				},
			},
			Port: 80,
		}),
	)

	config := &files.Config{
		Name:      projectName,
		GenConfig: genCfg,
	}
	config.InitGeneration(outputDir, projectName)
}

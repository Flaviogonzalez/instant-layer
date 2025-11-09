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
					Driver:      "mysql",
					URL:         "",
				},
				RoutesConfig: files.RoutesConfig{
					RoutesGroup: []*files.RoutesGroup{
						&files.RoutesGroup{
							Routes: []*files.Route{
								&files.Route{
									Path:   "/lol",
									Method: "POST",
									Handler: files.Handler{
										Name: "lol",
									},
								},
							},
						},
					},
				},
			},
			Port: 80,
		}),
		// files.AddToPackage("routes", files.AuthRoutesFile),
		// files.AddToPackage("handlers", files.AuthHandlerFile),
	)

	config := &files.Config{
		Name:      projectName,
		GenConfig: genCfg,
	}
	config.InitGeneration(outputDir, projectName)
}

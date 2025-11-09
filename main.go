package main

import (
	"instant-layer/files"
)

func main() {
	outputDir := "dist"
	projectName := "instant-layer"

	config := &files.Config{
		Name: projectName,
		GenConfig: files.GenConfig{
			Services: []files.Service{
				{
					Name:       "auth-service",
					ServerType: &files.API{},
				},
			},
		},
	}

	config.InitGeneration(outputDir, projectName)
}

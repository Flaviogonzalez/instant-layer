package main

import (
	"instant-layer/files"
)

func main() {
	outputDir := "dist"
	projectName := "instant-layer"

	config := &files.Config{
		Name: projectName,
		GenConfig: &files.GenConfig{
			Services: []files.Service{},
		},
	}

	config.InitGeneration(outputDir, projectName)
}

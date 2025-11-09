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
				{Name: "article-service"},
			},
		},
	}

	config.InitGeneration(outputDir, projectName)
}

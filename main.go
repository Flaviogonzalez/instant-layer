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
					Name: "article-service",
					Connections: []*files.Connection{
						&files.Connection{
							Type: &files.GRPC{},
						},
						&files.Connection{
							Type: &files.RPC{},
						},
					},
				},
			},
		},
	}

	config.InitGeneration(outputDir, projectName)
}

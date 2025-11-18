package cmd

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/flaviogonzalez/instant-layer/internal/files"

	"github.com/spf13/cobra"
)

func Do(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	rootCmd := &cobra.Command{Use: "layer", SilenceUsage: true}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateCmd)
	// unit-tests
	rootCmd.SetIn(stdin)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an instant.json settings file in the current directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		resolvedPath := filepath.Join(wd, "instant.json")

		f, err := os.Create(resolvedPath)
		if err != nil {
			return err
		}
		defer f.Close()

		c := definitionConfig("myproject")
		json, err := json.MarshalIndent(&c, "", "  ")
		if err != nil {
			return err
		}
		f.Write(json)

		return nil
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate [output]",
	Short: "Genera el proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] != "" && len(args) > 1 {
			log.Printf("Output file: %s", args[0])
		}
		data, err := os.ReadFile("instant.json")
		if err != nil {
			return err
		}

		var cfg files.Config
		json.Unmarshal(data, &cfg)

		cfg.InitGeneration(args[0])
		return nil
	},
}

func definitionConfig(projectName string) *files.Config {
	genCfg := files.NewGenConfig(
		files.WithService(files.Service{
			Name: "myservice",
			ServerType: &files.API{
				DB: files.Database{
					TimeoutConn: 15,
					Driver:      "pgx",
					URL:         "",
				},
				RoutesConfig: files.RoutesConfig{
					RoutesGroup: []*files.RoutesGroup{},
				},
			},
			Port: 80,
		}),
	)

	config := &files.Config{
		Name:      projectName,
		GenConfig: genCfg,
	}
	return config
}

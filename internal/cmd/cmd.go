package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/flaviogonzalez/instant-layer/files"

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

		c := definitionConfig("")
		json, err := json.Marshal(&c)
		f.Write(json)

		return nil
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate [output]",
	Short: "Genera el proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, _ := os.ReadFile("instant.json")

		var cfg files.Config
		json.Unmarshal(data, &cfg)

		cfg.Generate(args[0])
		return nil
	},
}

func definitionConfig(projectName string) *files.Config {
	genCfg := files.NewGenConfig(
		files.WithService(files.Service{
			Name: "",
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

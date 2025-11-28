package cmd

import (
	"io"
	"os"

	"github.com/flaviogonzalez/instant-layer/internal/config"
	"github.com/spf13/cobra"
)

func Do(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	rootCmd := &cobra.Command{
		Use:          "layer",
		Short:        "Instant Layer — Generador de servicios y arquitectura en Go",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(hydrateCmd)
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

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "creates root directory.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		var dir string
		if len(args) == 1 {
			dir = args[0]
		}

		return StartGeneration(wd, dir)
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "creates a new service, route, handler, middleware",
}

var addServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "creates a new service",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return nil
		}
		return SelectAndGenerateTemplate(dir)
	},
}

var hydrateCmd = &cobra.Command{
	Use:   "hydrate",
	Short: "project analysis and regeneration",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		root, err := config.FindLayerRoot(wd)
		if err != nil {
			return err
		}

		l := &config.Layer{Root: root}
		err = l.Reload()
		if err != nil {
			return err
		}

		err = l.Hydrate()
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	addCmd.AddCommand(addServiceCmd)
}

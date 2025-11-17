package cmd

import (
	"os"

	"github.com/instant-layer/instant-layer/internal/cmd"
)

func main() {
	os.Exit(cmd.Do(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

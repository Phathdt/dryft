package main

import (
	"context"
	"fmt"
	"os"

	"github.com/phathdt/dryft/internal/cli"
)

func main() {
	if err := cli.NewApp().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

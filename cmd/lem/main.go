package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	app := newApp(os.Stdout, os.Stderr)
	if err := app.Run(ctx, os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s %v\n", red("ERROR"), err)
		os.Exit(1)
	}
}

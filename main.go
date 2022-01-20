package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/bep/punused/internal/lib"
)

func main() {
	// Default to "every go file in the workspace".
	pattern := "**/*.go"
	if len(os.Args) > 1 {
		pattern = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	wd, _ := os.Getwd()

	err := lib.Run(
		ctx,
		lib.RunConfig{
			WorkspaceDir:    wd,
			FilenamePattern: pattern,
			Out:             os.Stdout,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}

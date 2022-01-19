package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/bep/unused/internal/lib"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: unused <glob pattern>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	wd, _ := os.Getwd()

	err := lib.Run(
		ctx,
		lib.RunConfig{
			WorkspaceDir:    wd,
			FilenamePattern: os.Args[1],
			Out:             os.Stdout,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}

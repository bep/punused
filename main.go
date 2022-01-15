package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bep/unused/internal/lib"
	"github.com/gobwas/glob"
)

var (
	colorUnused   = "\033[32m"
	colorTestOnly = "\033[34m"
	colorReset    = "\033[0m"
)

func init() {
	if runtime.GOOS == "windows" {
		colorTestOnly = ""
		colorUnused = ""
		colorReset = ""
	}

	// Used in test.
	_ = lib.UsedVar
}

func main() {
	r, err := Start(os.Args)
	exitOnErr("Failed to start gopls", err)
	defer func() {
		exitOnErr("Failed to stop gopls", r.Stop())
	}()
	exitOnErr("Failed to walk", r.Walk())
}

type runner struct {
	wd          string
	filematcher glob.Glob
	client      *lib.GoplsClient
}

func (r *runner) Stop() error {
	return r.client.Close()
}

func Start(args []string) (*runner, error) {
	if len(args) != 2 {
		return nil, errors.New("Usage: unused <glob file pattern (double asterisk supported)>")
	}

	filenamePattern := args[1]
	matcher, err := glob.Compile(filenamePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %s", err)
	}

	wd, _ := os.Getwd()

	client, err := lib.NewClient(wd)
	if err != nil {
		return nil, err
	}

	return &runner{client: client, wd: wd, filematcher: matcher}, nil
}

func (r *runner) Walk() error {
	return filepath.Walk(r.wd, func(path string, info fs.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		base := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, r.wd)), "/")

		if !r.filematcher.Match(base) {
			return nil
		}

		r.handleFile(base)
		return nil
	})
}

func (r *runner) handleFile(filename string) {
	if strings.HasSuffix(filename, "_test.go") {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	symbols, err := r.client.DocumentSymbol(ctx, filename)
	exitOnErr("Failed to get symbols", err)

	for _, s := range symbols {
		if !isExported(s.Name) {
			continue
		}

		refs, err := r.client.DocumentReferences(ctx, s.Location)
		exitOnErr("Failed to get references", err)

		var unused bool
		var testOnly bool
		if len(refs) == 0 {
			unused = true
		} else {
			testOnly = true
			for _, ref := range refs {
				if !strings.HasSuffix(string(ref.URI), "_test.go") {
					testOnly = false
					break
				}
			}
		}

		if unused || testOnly {
			e := Usage{
				Filename:   filename,
				Symbol:     s,
				IsTestOnly: testOnly,
			}
			fmt.Println(e.String())
		}
	}
}

func exitOnErr(msg string, err error) {
	if err != nil {
		fmt.Println(msg)
		log.Fatalf("%s: %s", msg, err)
	}
}

type UnusedStruct struct {
	Name string
}

type Usage struct {
	Filename   string
	Symbol     *lib.Symbol
	IsTestOnly bool
}

func (u Usage) String() string {
	s := u.Symbol
	loc := s.Location
	kind := strings.ToLower(string(s.Kind.String()))
	line, col := loc.Range.Start.Line+1, loc.Range.Start.Character+1

	if u.IsTestOnly {
		msg := fmt.Sprintf("%s:%d:%d %s %s is used in test only (EU1001)", u.Filename, line, col, kind, s.Name)
		return (colorTestOnly + msg + colorReset)
	}
	msg := fmt.Sprintf("%s:%d:%d %s %s is unused (EU1002)", u.Filename, line, col, kind, s.Name)
	return (colorUnused + msg + colorReset)
}

func isExported(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
}

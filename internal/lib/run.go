package lib

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
	"github.com/sourcegraph/go-lsp"
)

func Run(ctx context.Context, cfg RunConfig) (err error) {
	if err := cfg.validate(); err != nil {
		return err
	}

	// This needs to be run from the rooot of a Go Module to get correct results.
	if _, err := os.Stat(filepath.Join(cfg.WorkspaceDir, "go.mod")); err != nil {
		return fmt.Errorf("workspace %s is not a Go module (go.mod is missing): %w", cfg.WorkspaceDir, err)
	}

	r, err := newRunner(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		err = r.Stop()
	}()

	err = r.Walk()

	return
}

func newRunner(ctx context.Context, cfg RunConfig) (*runner, error) {
	matcher, err := glob.Compile(cfg.FilenamePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	client, err := newClient(ctx, cfg.WorkspaceDir)
	if err != nil {
		return nil, err
	}

	return &runner{ctx: ctx, client: client, cfg: cfg, filematcher: matcher}, nil
}

type RunConfig struct {
	WorkspaceDir    string
	FilenamePattern string
	Out             io.Writer
}

func (cfg RunConfig) validate() error {
	if cfg.WorkspaceDir == "" {
		return fmt.Errorf("WorkspaceDir is required")
	}
	if cfg.FilenamePattern == "" {
		return fmt.Errorf("FilenamePattern is required")
	}
	if cfg.Out == nil {
		return fmt.Errorf("Out is required")
	}
	return nil
}

type runner struct {
	ctx         context.Context
	cfg         RunConfig
	filematcher glob.Glob
	client      *GoplsClient
}

func (r *runner) Stop() error {
	return r.client.Close()
}

func (r *runner) Walk() error {
	return filepath.Walk(r.cfg.WorkspaceDir, func(path string, info fs.FileInfo, err error) error {
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

		base := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, r.cfg.WorkspaceDir)), "/")

		if !r.filematcher.Match(base) {
			return nil
		}

		return r.handleFile(base)
	})
}

func (r *runner) handleFile(filename string) error {
	if strings.HasSuffix(filename, "_test.go") {
		return nil
	}

	symbols, err := r.client.DocumentSymbol(r.ctx, filename)
	if err != nil {
		return fmt.Errorf("failed to get symbols: %w", err)
	}

	var handleSymbol func(s *Symbol) error
	handleSymbol = func(s *Symbol) error {
		base := s.Name
		if s.Kind == lsp.SKMethod && strings.Contains(base, ".") {
			// Struct methods' Name comes on the form  (MyType).MyMethod.
			base = s.Name[strings.Index(s.Name, ".")+1:]
		}

		if !isExported(base) {
			return nil
		}

		refs, err := r.client.DocumentReferences(r.ctx, s.Location)
		if err != nil {
			return fmt.Errorf("failed to get references: %w", err)
		}

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
			e := usage{
				Filename:   filename,
				Symbol:     s,
				IsTestOnly: testOnly,
			}
			e.Print(r.cfg.Out)
		}

		for _, child := range s.Children {
			if err := handleSymbol(child); err != nil {
				return err
			}
		}

		return nil
	}

	for _, s := range symbols {
		if err := handleSymbol(s); err != nil {
			return err
		}
	}

	return nil
}

type usage struct {
	Filename   string
	Symbol     *Symbol
	IsTestOnly bool
}

func (u usage) Print(w io.Writer) {
	s := u.Symbol
	loc := s.Location
	kind := strings.ToLower(string(s.Kind.String()))
	line, col := loc.Range.Start.Line+1, loc.Range.Start.Character+1
	if u.IsTestOnly {
		fmt.Fprintf(w, "%s:%d:%d %s %s is used in test only (EU1001)\n", u.Filename, line, col, kind, s.Name)
	} else {
		fmt.Fprintf(w, "%s:%d:%d %s %s is unused (EU1002)\n", u.Filename, line, col, kind, s.Name)
	}
}

func isExported(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
}

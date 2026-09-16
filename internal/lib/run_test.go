package lib

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/context"
)

func TestRun(t *testing.T) {
	c := qt.New(t)

	var buff bytes.Buffer

	// The WorkDir needs to be a the module (workspace) root.
	wd, _ := os.Getwd()
	wd = filepath.Join(wd, "..", "..")

	c.Assert(
		Run(
			context.Background(),
			RunConfig{
				WorkspaceDir:    wd,
				FilenamePattern: "**/testpackages/**.go",
				Out:             &buff,
			},
		),
		qt.IsNil,
	)

	golden := `
internal/lib/testpackages/firstpackage/code1.go:7:2 variable UnusedVar is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:12:2 constant UnusedConst is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:19:6 function UnusedFunction is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:25:2 field UnusedField is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:32:15 method (MyType).UnusedMethod is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:36:6 interface UnusedInterfaceWithUsedAndUnusedMethod is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:37:2 method UsedInterfaceMethodReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:38:2 method UnusedInterfaceMethodReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:41:6 interface UnusedInterface is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:42:2 method UnusedInterfaceReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:45:6 interface UsedInterface is unused (EU1002)
internal/lib/testpackages/firstpackage/testlib1.go:4:2 constant OnlyUsedInTestConst is used in test only (EU1001)
`

	if diff := cmp.Diff(strings.TrimSpace(buff.String()), strings.TrimSpace(golden)); diff != "" {
		c.Fatal("unexpected output\n", diff+"\n\n"+buff.String())
	}
}

func TestWalkReturnsWalkError(t *testing.T) {
	c := qt.New(t)

	r := &runner{
		cfg: RunConfig{
			WorkspaceDir:    filepath.Join(t.TempDir(), "does-not-exist"),
			FilenamePattern: "**/*.go",
			Out:             &bytes.Buffer{},
		},
	}

	c.Assert(r.Walk(), qt.ErrorIs, fs.ErrNotExist)
}

// cancelOnFinding cancels on the first finding printed, where no request is in
// flight, so a later one fails mid-walk.
type cancelOnFinding struct{ cancel func() }

func (w *cancelOnFinding) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }

func TestRunKeepsWalkErrorOverStop(t *testing.T) {
	c := qt.New(t)

	wd, _ := os.Getwd()
	wd = filepath.Join(wd, "..", "..")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := Run(ctx, RunConfig{
		WorkspaceDir:    wd,
		FilenamePattern: "**/testpackages/**.go",
		Out:             &cancelOnFinding{cancel: cancel},
	})

	c.Assert(err, qt.ErrorIs, context.Canceled)
}

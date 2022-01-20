package lib

import (
	"bytes"
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
internal/lib/testpackages/firstpackage/code1.go:38:2 method UnusedInterfaceMethodReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:37:2 method UsedInterfaceMethodReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:41:6 interface UnusedInterface is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:42:2 method UnusedInterfaceReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:45:6 interface UsedInterface is unused (EU1002)
`

	if diff := cmp.Diff(strings.TrimSpace(buff.String()), strings.TrimSpace(golden)); diff != "" {
		c.Fatal("unexpected output\n", diff+"\n\n"+buff.String())
	}
}

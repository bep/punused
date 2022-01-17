package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"golang.org/x/net/context"
)

var _ = OnlyUsedInTestVar

func TestClient(t *testing.T) {
	c := qt.New(t)

	dir, _ := os.Getwd()
	workDir := filepath.Join(dir, "..", "..")

	client, err := NewClient(workDir)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	symbols, err := client.DocumentSymbol(ctx, "internal/lib/gopls.go")
	c.Assert(err, qt.IsNil)
	var selectedSymb *Symbol
	for _, symb := range symbols {
		if symb.Name == "UsedVar" {
			selectedSymb = symb
		}
	}
	c.Assert(selectedSymb, qt.IsNotNil)
	refs, err := client.DocumentReferences(ctx, selectedSymb.Location)
	c.Assert(err, qt.IsNil)
	for _, ref := range refs {
		fmt.Printf("\t%s:%d\n", ref.URI, ref.Range.Start.Line+1)
	}
	c.Assert(len(refs), qt.Equals, 1)
	uri := string(refs[0].URI)
	c.Assert(strings.HasSuffix(uri, "main.go"), qt.IsTrue, qt.Commentf("%s", uri))
}

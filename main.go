package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: unused <glob file pattern or package selector>")
	}

	filenamePattern := os.Args[1]
	r := &runner{}

	if strings.HasSuffix(filenamePattern, "/...") {
		root := filepath.FromSlash(strings.TrimSuffix(filenamePattern, "/..."))
		fmt.Println("Scanning", root)
		err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if info != nil && info.IsDir() {
				return nil
			}
			r.handleFile(path)
			return nil
		})
		must(err, "Error walking")
		return
	}

	filenames, err := filepath.Glob(filenamePattern)
	must(err, "failed to glob")

	for _, filename := range filenames {
		r.handleFile(filename)
	}
}

type runner struct{}

func (r *runner) handleFile(filename string) {
	if strings.HasSuffix(filename, "_test.go") {
		return
	}
	symbols := r.getSymbols(filename)
	for _, s := range symbols {
		if !isExported(s.Name) {
			continue
		}

		refs := r.getReferences(filename, s.Range.Start)
		var unused bool
		var testOnly bool
		if len(refs) == 0 {
			unused = true
		} else {
			testOnly = true
			for _, ref := range refs {
				if !ref.IsTest {
					testOnly = false
					break
				}
			}
		}

		if unused || testOnly {
			e := Usage{
				Message: fmt.Sprintf("%s:%s %s %s", filename, s.Range.Start, s.Name, s.Type),
				IsTest:  testOnly,
			}
			fmt.Println(e.String())
		}
	}
}

func (r *runner) getSymbols(filename string) []Symbol {
	symbolsList := r.runGopls("symbols", filename)
	lines := strings.Split(symbolsList, "\n")

	var symbols []Symbol
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			panic(fmt.Sprintf("Unexpected line: %s", line))
		}
		startEnd := strings.Split(parts[2], "-")
		if len(startEnd) != 2 {
			panic(fmt.Sprintf("Unexpected position: %s", startEnd))
		}

		symbols = append(symbols, Symbol{
			Name: parts[0],
			Type: parts[1],
			Range: Range{
				Start: startEnd[0],
				End:   startEnd[1],
			},
		})

	}

	return symbols
}

func (r *runner) getReferences(filename, pos string) []Reference {
	referencesList := r.runGopls("references", filename+":"+pos)

	lines := strings.Split(strings.TrimSpace(referencesList), "\n")

	var references []Reference
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		references = append(references, Reference{
			Ref:    line,
			IsTest: strings.Contains(line, "_test.go"),
		})

	}
	return references
}

func (r *runner) runGopls(feature string, args ...string) string {
	args = append([]string{feature}, args...)
	b, err := exec.Command("gopls", args...).CombinedOutput()
	must(err, string(b))
	return string(b)
}

func must(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		log.Fatal(err)
	}
}

type Symbol struct {
	Name  string
	Type  string
	Range Range
}

type Range struct {
	Start string
	End   string
}

type Reference struct {
	Ref    string
	IsTest bool
}

type Usage struct {
	Message string
	IsTest  bool
}

func (u Usage) String() string {
	color := colorUnused
	prefix := "Unused: "
	if u.IsTest {
		color = colorTestOnly
		prefix = "Test only: "
	}

	return (color + prefix + u.Message + colorReset)
}

func isExported(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
}

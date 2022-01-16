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
		r.printResult()
		return
	}

	filenames, err := filepath.Glob(filenamePattern)
	must(err, "failed to glob")

	for _, filename := range filenames {
		r.handleFile(filename)
	}

	r.printResult()
}

type runner struct {
	unused          []string
	usedInTestsOnly []string
}

func (r *runner) printResult() {
	fmt.Println()
	fmt.Print(colorTestOnly + "Used only in tests:" + colorReset + "\n\n")

	for _, e := range r.usedInTestsOnly {
		fmt.Println(colorTestOnly + e + colorReset)
	}

	fmt.Println()
	fmt.Print(colorUnused + "Unused:" + colorReset + "\n\n")

	for _, e := range r.unused {
		fmt.Println(colorUnused + e + colorReset)
	}
}

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
		if len(refs) == 0 {
			r.unused = append(r.unused, fmt.Sprintf("%s:%s:%s:%s", filename, s.Range.Start, s.Type, s.Name))
		} else {
			var nonTestUsage bool
			for _, ref := range refs {
				if !ref.IsTest {
					nonTestUsage = true
					break
				}
			}

			if !nonTestUsage {
				r.usedInTestsOnly = append(r.usedInTestsOnly, fmt.Sprintf("%s:%s:%s:%s", filename, s.Range.Start, s.Type, s.Name))
			}
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
		startEnd := strings.Split(parts[2], "-")

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

type Usage struct {
	Filename string
	Symbol   Symbol
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

func isExported(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
}

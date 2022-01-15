package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: unused \"<glob file pattern>\"")
	}

	filenamePattern := os.Args[1]

	filenames, err := filepath.Glob(filenamePattern)
	must(err, "failed to glob")

	for _, filename := range filenames {
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		symbols := getSymbols(filename)
		for _, s := range symbols {
			if isExported(s.Name) {
				refs := getReferences(filename, s.Range.Start)
				if len(refs) == 0 {
					// Unused
					fmt.Printf("%s:%s:%s:%s\n", filename, s.Range.Start, s.Type, s.Name)
				}
			}
		}

	}
}

func getSymbols(filename string) []Symbol {
	symbolsList := runGopls("symbols", filename)
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

func getReferences(filename, pos string) []string {
	referencesList := runGopls("references", filename+":"+pos)

	lines := strings.Split(strings.TrimSpace(referencesList), "\n")

	var references []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Skip test references.
		if strings.Contains(line, "_test.go") {
			continue
		}

		references = append(references, line)

	}
	return references
}

func runGopls(feature string, args ...string) string {
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

func isExported(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
}

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pclk/anki.go/cmd"
	"github.com/pclk/anki.go/fileops"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cleanupFlag := flag.Bool("c", false, "Cleanup apy-*.md files")
	defaultDeck := flag.String("d", "", "Specify default deck for import")
	flag.Parse()

	if *cleanupFlag {
		return fileops.RemoveOutputFile()
	}

	if flag.NArg() != 1 {
		return fmt.Errorf("Usage: anki <input_file.md> [-d default_deck]")
	}

	inputFile := flag.Arg(0)
	return cmd.CreateNote(inputFile, *defaultDeck)
}

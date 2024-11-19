package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pclk/anki.go/anthropic"
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

	askClaude := flag.Bool("ask", false, "Ask Claude a question")
	apiKey := flag.String("key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
	model := flag.String("model", "claude-3-5-sonnet-20241022", "Claude model to use")
	conversationFile := flag.String("conv", "", "Conversation file to use/create")
	pdfPath := flag.String("pdf", "", "Path to PDF to analyze")
	flag.Parse()

	if *cleanupFlag {
		return fileops.RemoveOutputFile()
	}

	if *askClaude {
		if flag.NArg() != 1 {
			return fmt.Errorf("Usage with -ask: provide your question as an argument")
		}
		return anthropic.ContinueConversation(flag.Arg(0), *pdfPath, *apiKey, *model, *conversationFile)
	}

	if flag.NArg() != 1 {
		return fmt.Errorf("Usage: anki [-d default_deck] <input_file.md> ")
	}

	inputFile := flag.Arg(0)
	return cmd.CreateNote(inputFile, *defaultDeck)
}

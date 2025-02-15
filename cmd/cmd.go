package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pclk/anki.go/converter"
)

func CreateNote(inputFile, defaultDeck string) error {
	// Get the directory and filename separately
	dir, file := filepath.Split(inputFile)

	// Create the output filename with "apy-" prefix
	outputFile := filepath.Join(dir, "anki-"+file)

	deckName, err := converter.ConvertToAnki(inputFile, outputFile)
	if err != nil {
		return err
	}
	fmt.Println("deckName", deckName)

	fmt.Printf("Conversion complete. Output file: %s\n", outputFile)

	command := []string{"apy", "add-from-file", outputFile}
	if defaultDeck != "" {
		command = append(command, "-d", defaultDeck)
	} else if deckName != "" {
		command = append(command, "-d", deckName)
	}

	output, err := ExecuteCommand(command...)
	if err != nil {
		return fmt.Errorf("error executing apy command: %v", err)
	}

	fmt.Println("apy command output:")
	fmt.Println(output)

	return nil
}

func ExecuteCommand(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("error executing command '%s': %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output), nil
}

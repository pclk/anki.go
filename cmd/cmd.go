package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/pclk/anki.go/converter"
)

func CreateNote(inputFile, defaultDeck string) error {
	// Get the directory and filename separately
	dir, file := filepath.Split(inputFile)

	// Create the output filename with "apy-" prefix
	outputFile := filepath.Join(dir, "anki-"+file)

	if err := converter.ConvertToAnki(inputFile, outputFile); err != nil {
		return err
	}

	fmt.Printf("Conversion complete. Output file: %s\n", outputFile)

	command := []string{"apy", "add-from-file", outputFile}
	if defaultDeck != "" {
		command = append(command, "-d", defaultDeck)
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
	return string(output), err
}

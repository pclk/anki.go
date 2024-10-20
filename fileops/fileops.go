package fileops

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

func OpenInputFile(filename string) (*bufio.Scanner, func(), error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		file.Close()
	}
	return bufio.NewScanner(file), cleanup, nil
}

func CreateOutputFile(filename string) (*bufio.Writer, func(), error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		file.Close()
	}
	return bufio.NewWriter(file), cleanup, nil
}

func RemoveOutputFile() error {
	files, err := filepath.Glob("apy-*.md")
	if err != nil {
		return fmt.Errorf("error finding files to clean up: %v", err)
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("error removing file %s: %v", file, err)
		}
		fmt.Printf("Removed file: %s\n", file)
	}

	return nil
}

package converter

import (
	"bufio"
	"strings"

	"github.com/pclk/anki.go/fileops"
)

func ConvertToAnki(inputFile, outputFile string) error {
	scanner, inputCleanup, err := fileops.OpenInputFile(inputFile)
	if err != nil {
		return err
	}
	defer inputCleanup()

	writer, outputCleanup, err := fileops.CreateOutputFile(outputFile)
	if err != nil {
		return err
	}
	defer outputCleanup()

	return processFile(scanner, writer)
}

func processFile(scanner *bufio.Scanner, writer *bufio.Writer) error {
	writer.WriteString("model: Basic\n\n")
	var front, back strings.Builder
	inFront := true

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			if front.Len() > 0 || back.Len() > 0 {
				writeNote(writer, front.String(), back.String())
				front.Reset()
				back.Reset()
			}
			front.WriteString(strings.TrimPrefix(line, "# "))
			inFront = false
		} else if line == "" {
			if inFront {
				front.WriteString("\n")
			} else {
				back.WriteString("\n")
			}
		} else {
			if inFront {
				front.WriteString("\n" + line)
			} else {
				back.WriteString("\n" + line)
			}
		}
	}

	if front.Len() > 0 || back.Len() > 0 {
		writeNote(writer, front.String(), back.String())
	}

	return writer.Flush()
}

func writeNote(writer *bufio.Writer, front, back string) {
	writer.WriteString("# Note\n\n")
	writer.WriteString("## Front\n")
	writer.WriteString(strings.TrimSpace(front) + "\n\n")
	writer.WriteString("## Back\n")
	writer.WriteString(strings.TrimSpace(back) + "\n\n")
}

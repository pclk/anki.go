package converter

import (
	"bufio"
	"fmt"
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
	clozeCounter := 0

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "# ") {
			if front.Len() > 0 || back.Len() > 0 {
				if strings.Contains(front.String(), "{{") {
					writeClozeDeletion(writer, front.String(), back.String(), &clozeCounter)
				} else {
					writeBasicNote(writer, front.String(), back.String())
				}
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

	// Handle the last note
	if front.Len() > 0 || back.Len() > 0 {
		if strings.Contains(front.String(), "{{") {
			writeClozeDeletion(writer, front.String(), back.String(), &clozeCounter)
		} else {
			writeBasicNote(writer, front.String(), back.String())
		}
	}

	return writer.Flush()
}

func writeBasicNote(writer *bufio.Writer, front, back string) {
	writer.WriteString("# Note\n\n")
	writer.WriteString("## Front\n")
	writer.WriteString(strings.TrimSpace(front) + "\n\n")
	writer.WriteString("## Back\n")
	writer.WriteString(strings.TrimSpace(back) + "\n\n")
}

func writeClozeDeletion(writer *bufio.Writer, text, backExtra string, counter *int) {
	writer.WriteString("# Note\n")
	writer.WriteString("model: Cloze\n\n")
	writer.WriteString("## Text\n")

	// Process cloze deletions using simple string operations
	processed := ""
	remaining := text
	clozeCounter := 0
	for {
		start := strings.Index(remaining, "{{")
		if start == -1 {
			processed += remaining
			break
		}

		end := strings.Index(remaining[start:], "}}")
		if end == -1 {
			processed += remaining
			break
		}
		end += start + 2

		clozeCounter++
		processed += remaining[:start]
		clozeText := remaining[start+2 : end-2]
		processed += fmt.Sprintf("{{c%d::%s}}", clozeCounter, clozeText)

		remaining = remaining[end:]
	}

	writer.WriteString(strings.TrimSpace(processed) + "\n\n")
	writer.WriteString("## Back Extra\n")
	writer.WriteString(strings.TrimSpace(backExtra) + "\n\n")
}

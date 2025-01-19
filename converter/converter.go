package converter

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/pclk/anki.go/fileops"
)

func ExtractDeckName(scanner *bufio.Scanner) string {
	if scanner.Scan() {
		firstLine := scanner.Text()
		firstLineLower := strings.ToLower(firstLine)
		if strings.HasPrefix(firstLineLower, "# deck:") {
			return strings.TrimSpace(strings.TrimPrefix(firstLine, firstLine[:7]))
		}
	}
	return ""
}

func ConvertToAnki(inputFile, outputFile string) (string, error) {
	scanner, inputCleanup, err := fileops.OpenInputFile(inputFile)
	if err != nil {
		return "", err
	}
	defer inputCleanup()

	// Read deck name first
	deckName := ExtractDeckName(scanner)

	// Reopen the file for processing
	scanner, inputCleanup, err = fileops.OpenInputFile(inputFile)
	if err != nil {
		return "", err
	}
	defer inputCleanup()

	writer, outputCleanup, err := fileops.CreateOutputFile(outputFile)
	if err != nil {
		return "", err
	}
	defer outputCleanup()

	if err := processFile(scanner, writer); err != nil {
		return "", err
	}
	return deckName, nil
}

func processFile(scanner *bufio.Scanner, writer *bufio.Writer) error {
	var front, back strings.Builder
	var currentSection string

	// Skip the deck specification line if it exists
	if scanner.Scan() {
		firstLine := scanner.Text()
		if !strings.HasPrefix(strings.ToLower(firstLine), "# deck:") {
			processLine(firstLine, &front, &back, writer, &currentSection)
		}
	}

	// Process remaining lines
	for scanner.Scan() {
		line := scanner.Text()
		processLine(line, &front, &back, writer, &currentSection)
	}

	// Write any remaining content
	if front.Len() > 0 || back.Len() > 0 {
		if hasClozeMarkers(front.String()) {
			writeClozeDeletion(writer, front.String(), back.String(), nil)
		} else {
			writeBasicNote(writer, front.String(), back.String())
		}
	}

	return writer.Flush()
}

type Section struct {
	Level   int
	Content string
}

var currentSections []Section

func processLine(line string, front, back *strings.Builder, writer *bufio.Writer, currentSection *string) {
	fmt.Printf("\n=== Processing line: %q\n", line)
	fmt.Printf("Current front buffer:\n%q\n", front.String())
	fmt.Printf("Current back buffer:\n%q\n", back.String())
	// Handle section headers
	if strings.HasPrefix(line, "#") {
		// Count the level by number of #
		level := 0
		for i, char := range line {
			if char == '#' {
				level++
			} else {
				// Remove sections at or below this level
				for len(currentSections) > 0 && currentSections[len(currentSections)-1].Level >= level {
					currentSections = currentSections[:len(currentSections)-1]
				}
				// Add new section
				sectionContent := strings.TrimSpace(line[i:])
				currentSections = append(currentSections, Section{
					Level:   level,
					Content: strings.TrimPrefix(sectionContent, " "),
				})
				break
			}
		}
		return
	}

	// Only write out note if we're starting a new section or note
	if (strings.HasPrefix(line, "#") || strings.HasSuffix(line, "?")) && (front.Len() > 0 || back.Len() > 0) {
		if hasClozeMarkers(front.String()) {
			writeClozeDeletion(writer, front.String(), back.String(), nil)
		} else {
			writeBasicNote(writer, front.String(), back.String())
		}
		front.Reset()
		back.Reset()
	}

	// Process the current line
	switch {
	case strings.HasPrefix(line, ">"):
		// Back extra for cloze
		if back.Len() > 0 {
			back.WriteString("\n")
		}
		back.WriteString(strings.TrimSpace(strings.TrimPrefix(line, ">")))

	case hasClozeMarkers(line):
		// Cloze deletion
		if len(currentSections) > 0 {
			for i, section := range currentSections {
				prefix := "Section: "
				if i > 0 {
					prefix = "Sub-section: "
				}
				front.WriteString(prefix + section.Content + "\n\n")
			}
			front.WriteString("\n") // Double newline after sections for better spacing
		}
		front.WriteString(line + "\n")

	case strings.HasSuffix(line, "?"):
		// Write out previous note if exists
		if front.Len() > 0 || back.Len() > 0 {
			if hasClozeMarkers(front.String()) {
				writeClozeDeletion(writer, front.String(), back.String(), nil)
			} else {
				writeBasicNote(writer, front.String(), back.String())
			}
			front.Reset()
			back.Reset()
		}

		// Start new question (front of basic card)
		if len(currentSections) > 0 {
			for i, section := range currentSections {
				prefix := "Section: "
				if i > 0 {
					prefix = "Sub-section: "
				}
				front.WriteString(prefix + section.Content + "\n\n")
			}
			front.WriteString("\n") // Double newline after sections for better spacing
		}
		front.WriteString(line)

	case line == "":
		// Empty line marks the end of a note
		if front.Len() > 0 || back.Len() > 0 {
			if hasClozeMarkers(front.String()) {
				writeClozeDeletion(writer, front.String(), back.String(), nil)
			} else {
				writeBasicNote(writer, front.String(), back.String())
			}
			front.Reset()
			back.Reset()
		}

	default:
		// If we have a question (front ends with ?), this is the answer
		if strings.HasSuffix(strings.TrimSpace(front.String()), "?") {
			if back.Len() > 0 {
				back.WriteString("\n")
			}
			back.WriteString(line)
		} else {
			// Otherwise add to front
			if front.Len() > 0 {
				front.WriteString("\n")
			}
			front.WriteString(line)
		}
	}
}

func writeBasicNote(writer *bufio.Writer, front, back string) {
	fmt.Printf("\n=== DEBUG: Writing Basic Note ===\n")
	fmt.Printf("Raw front being written:\n%q\n", front)
	fmt.Printf("Raw back being written:\n%q\n", back)
	writer.WriteString("# Note\n\n")
	writer.WriteString("## Front\n\n")
	writer.WriteString(front + "\n\n")
	writer.WriteString("## Back\n\n")
	writer.WriteString(back + "\n\n")
}

func hasClozeMarkers(text string) bool {
	return strings.Contains(text, "-") && strings.Count(text, "-")%2 == 0
}

func writeClozeDeletion(writer *bufio.Writer, text, backExtra string, counter *int) {
	fmt.Printf("\n=== DEBUG: Writing Cloze ===\n")
	fmt.Printf("Raw text being written:\n%q\n", text)
	fmt.Printf("Back extra:\n%q\n", backExtra)
	writer.WriteString("# Note\n")
	writer.WriteString("model: Cloze\n\n")
	writer.WriteString("## Text\n\n")
	// Process cloze deletions using simple string operations
	lines := strings.Split(text, "\n")
	clozeCounter := 0
	groupMap := make(map[string]int) // Maps group numbers to cloze numbers
	var processedLines []string

	for _, line := range lines {
		if line == "" {
			processedLines = append(processedLines, line)
			continue
		}

		processed := ""
		remaining := line

		for {
			start := strings.Index(remaining, "-")
			if start == -1 {
				processed += remaining
				break
			}

			end := strings.Index(remaining[start+1:], "-")
			if end == -1 {
				processed += remaining
				break
			}
			end += start + 1

			// Check for group number before the cloze
			groupNum := ""
			beforeCloze := remaining[:start]
			if len(beforeCloze) > 0 {
				lastDot := strings.LastIndex(beforeCloze, ".")
				if lastDot != -1 && lastDot > 0 {
					potentialNum := strings.TrimSpace(beforeCloze[lastDot-1 : lastDot])
					if _, err := fmt.Sscanf(potentialNum, "%s", &groupNum); err == nil {
						// Remove the group number from the output
						processed = processed[:len(processed)-2]
					}
				}
			}

			// Handle escaped hyphens
			clozeText := remaining[start+1 : end]
			if strings.Contains(clozeText, "\\-") {
				clozeText = strings.ReplaceAll(clozeText, "\\-", "-")
			}

			var clozeNum int
			if groupNum != "" {
				if num, exists := groupMap[groupNum]; exists {
					clozeNum = num
				} else {
					clozeCounter++
					clozeNum = clozeCounter
					groupMap[groupNum] = clozeNum
				}
			} else {
				clozeCounter++
				clozeNum = clozeCounter
			}

			processed += remaining[:start]
			processed += fmt.Sprintf("{{c%d::%s}}", clozeNum, clozeText)

			remaining = remaining[end+1:]
		}

		processedLines = append(processedLines, processed)
	}

	processedText := strings.Join(processedLines, "\n")
	writer.WriteString(processedText + "\n")
	writer.WriteString("## Back Extra\n\n")
	if backExtra != "" {
		writer.WriteString(backExtra + "\n\n")
	}
	writer.WriteString("\n")
}

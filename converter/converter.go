package converter

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/pclk/anki.go/fileops"
)

func ExtractDeckName(scanner *bufio.Scanner) string {
	// Scan until we find the deck line
	for scanner.Scan() {
		line := scanner.Text()
		lineLower := strings.ToLower(line)
		if strings.HasPrefix(lineLower, "# deck:") {
			return strings.TrimSpace(strings.TrimPrefix(line, line[:7]))
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
	foundDeck := false
	previousLine = ""

	// Skip everything until we find the deck line
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.ToLower(line), "# deck:") {
			foundDeck = true
			break
		}
	}

	if !foundDeck {
		return fmt.Errorf("no deck specification found")
	}

	// Process remaining lines
	for scanner.Scan() {
		line := scanner.Text()
		if err := processLine(line, &front, &back, writer, &currentSection, scanner); err != nil {
			return err
		}
	}

	// Check for isolated line at the end of file
	if front.Len() > 0 {
		frontContent := front.String()
		if previousLine == "" && !hasClozeMarkers(frontContent) && !strings.HasSuffix(frontContent, "?") {
			fmt.Printf("\n⚠️  WARNING: Isolated line at end of file without cloze markers or question mark:\n%s\n", frontContent)
			fmt.Print("Continue processing? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return fmt.Errorf("process stopped by user")
			}
		}
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

// Track the previous line to detect isolated lines
var previousLine string

func processLine(line string, front, back *strings.Builder, writer *bufio.Writer, currentSection *string, scanner *bufio.Scanner) error {
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
		return nil
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
				front.WriteString(prefix + section.Content + "\n")
			}
			front.WriteString("\n")
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
			// Check for isolated lines without markers
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" && !hasClozeMarkers(line) && !strings.HasSuffix(line, "?") {
				// Check if this line is isolated (blank lines before and after)
				if previousLine == "" && front.Len() == 0 {
					// Store this line to check if the next line is blank
					if scanner.Scan() {
						nextLine := scanner.Text()
						if nextLine == "" {
							fmt.Printf("\n⚠️  WARNING: Isolated line without cloze markers or question mark:\n%s\n", line)
							fmt.Print("Continue processing? (y/N): ")
							var response string
							fmt.Scanln(&response)
							if response != "y" && response != "Y" {
								return fmt.Errorf("process stopped by user")
							}
						}
						// Process this line in the next iteration
						previousLine = line
						line = nextLine
					}
				}
			}

			// Add to front anyway
			if front.Len() > 0 {
				front.WriteString("\n")
			}
			front.WriteString(line)
		}
	}
	return nil
}

func writeBasicNote(writer *bufio.Writer, front, back string) {
	fmt.Printf("\n=== DEBUG: Writing Basic Note ===\n")
	fmt.Printf("Raw front being written:\n%q\n", front)
	fmt.Printf("Raw back being written:\n%q\n", back)
	writer.WriteString("# Note\n\n")
	writer.WriteString("## Front\n\n")

	// Add extra newline for each newline in front content
	frontLines := strings.Split(front, "\n")
	for i, line := range frontLines {
		writer.WriteString(line)
		if i < len(frontLines)-1 {
			writer.WriteString("\n\n") // Double newline between lines
		}
	}
	writer.WriteString("\n\n")

	writer.WriteString("## Back\n\n")

	// Add extra newline for each newline in back content
	backLines := strings.Split(back, "\n")
	for i, line := range backLines {
		writer.WriteString(line)
		if i < len(backLines)-1 {
			writer.WriteString("\n\n") // Double newline between lines
		}
	}
	writer.WriteString("\n\n")
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

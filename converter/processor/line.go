package processor

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/pclk/anki.go/converter/note"
	"github.com/pclk/anki.go/converter/template"
	"github.com/pclk/anki.go/converter/types"
)

// Track the previous line to detect isolated lines
var previousLine string

func ProcessLine(line string, front, back *strings.Builder, writer *bufio.Writer, currentSection *string, sections []types.Section, scanner *bufio.Scanner, tp *template.TemplateProcessor) error {
	fmt.Printf("Current state:\n")
	fmt.Printf("  Front buffer length: %d\n", front.Len())
	fmt.Printf("  Front buffer content: %q\n", front.String())
	fmt.Printf("  Back buffer length: %d\n", back.Len())
	fmt.Printf("  Back buffer content: %q\n", back.String())
	fmt.Printf("Checking for section header...\n")
	if strings.HasPrefix(line, "#") {
		fmt.Printf("  Found section header\n")
		// Write out any existing note before changing sections
		if front.Len() > 0 || back.Len() > 0 {
			if note.IsClozeFront(front.String()) {
				note.WriteCloze(writer, front.String(), back.String(), nil, sections)
			} else {
				note.WriteBasic(writer, front.String(), back.String())
			}
			front.Reset()
			back.Reset()
		}

		// Count the level by number of #
		level := 0
		for i, char := range line {
			if char == '#' {
				level++
			} else {
				fmt.Printf("  Section level: %d\n", level)
				// Clear all sections if we're at level 1 or 2 or if section is titled "Clear section"
				currentContent := strings.TrimSpace(line[i:])
				if level <= 2 || strings.TrimSpace(currentContent) == "Clear section" {
					fmt.Printf("  Clearing all sections (level <= 2 or Clear section)\n")
					sections = nil
				} else {
					fmt.Printf("  Removing sections at or below level %d\n", level)
					// Remove sections at or below this level
					for len(sections) > 0 && sections[len(sections)-1].Level >= level {
						fmt.Printf("    Removing section: %s (level %d)\n",
							sections[len(sections)-1].Content,
							sections[len(sections)-1].Level)
						sections = sections[:len(sections)-1]
					}
				}
				// Add new section
				sectionContent := strings.TrimSpace(line[i:])
				// Skip adding "Clear section" to the sections list
				if sectionContent != "Clear section" {
					fmt.Printf("  Adding new section: %s (level %d)\n", sectionContent, level)
					sections = append(sections, types.Section{
						Level:   level,
						Content: strings.TrimPrefix(sectionContent, " "),
					})
				}
				break
			}
		}
		return nil
	}

	// Only write out note if we're starting a new section or note
	if (strings.HasPrefix(line, "#") || note.IsBasicFront(line)) && (front.Len() > 0 || back.Len() > 0) {
		if note.IsClozeFront(front.String()) {
			note.WriteCloze(writer, front.String(), back.String(), nil, sections)
		} else {
			note.WriteBasic(writer, front.String(), back.String())
		}
		front.Reset()
		back.Reset()
	}

	// Process the current line
	switch {
	case strings.HasPrefix(line, ">"):
		fmt.Printf("  Processing back extra for cloze\n")
		if back.Len() > 0 {
			back.WriteString("\n")
		}
		back.WriteString(strings.TrimSpace(strings.TrimPrefix(line, ">")))

	case note.IsClozeFront(line):
		fmt.Printf("  Found cloze markers in line\n")
		if front.Len() > 0 || back.Len() > 0 {
			if note.IsClozeFront(front.String()) {
				note.WriteCloze(writer, front.String(), back.String(), nil, sections)
			} else {
				note.WriteBasic(writer, front.String(), back.String())
			}
			front.Reset()
			back.Reset()
		}

		// Start new cloze deletion
		front.WriteString(line)

	case note.IsBasicFront(line):
		fmt.Printf("  Found question mark suffix - starting new basic card\n")
		if front.Len() > 0 || back.Len() > 0 {
			if note.IsClozeFront(front.String()) {
				note.WriteCloze(writer, front.String(), back.String(), nil, sections)
			} else {
				note.WriteBasic(writer, front.String(), back.String())
			}
			front.Reset()
			back.Reset()
		}

		// Start new question (front of basic card)
		if len(sections) > 0 {
			mainSectionWritten := false
			for i, section := range sections {
				if section.Level <= 2 && !mainSectionWritten {
					front.WriteString("Section: " + section.Content + "\n\n")
					mainSectionWritten = true
				} else if section.Level > 2 && mainSectionWritten {
					if i == len(sections)-1 {
						if !strings.Contains(front.String(), "Sub-section:") {
							front.WriteString("Sub-section: " + section.Content + "\n")
						} else {
							front.WriteString(section.Content + "\n")
						}
					} else {
						if !strings.Contains(front.String(), "Sub-section:") {
							front.WriteString("Sub-section: " + section.Content + " > ")
						} else {
							front.WriteString(section.Content + " > ")
						}
					}
				}
			}
			front.WriteString("\n")
		}
		front.WriteString(line)

	default:
		fmt.Printf("  Processing default case\n")
		trimmedLine := strings.TrimSpace(line)

		// Handle basic card answer
		if strings.HasSuffix(strings.TrimSpace(front.String()), "?") {
			fmt.Printf("    Adding line as answer to basic card\n")
			if back.Len() > 0 {
				back.WriteString("\n")
			}
			back.WriteString(line)
			return nil
		}

		// Check for isolated lines
		if isIsolatedLine(trimmedLine, front.Len(), previousLine, tp) {
			fmt.Printf("    Found potential isolated line\n")
			if !scanner.Scan() {
				// EOF reached - this is definitely isolated
				fmt.Printf("\n⚠️  WARNING: Isolated line at end of file:\n%s\n", line)
				fmt.Print("Continue processing? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					return fmt.Errorf("process stopped by user")
				}
				return nil
			}

			nextLine := scanner.Text()
			if strings.TrimSpace(nextLine) == "" {
				fmt.Printf("WARNING: Isolated line found: %s\n", line)
				fmt.Print("Continue processing? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					return fmt.Errorf("process stopped by user")
				}
				return ProcessLine(nextLine, front, back, writer, currentSection, sections, scanner, tp)
			}

			// Process the next line since this wasn't actually isolated
			return ProcessLine(nextLine, front, back, writer, currentSection, sections, scanner, tp)
		}

		// Add line to front of card
		fmt.Printf("    Adding line to front of card\n")
		if front.Len() > 0 {
			front.WriteString("\n")
		}
		front.WriteString(line)
	}
	return nil
}

// isIsolatedLine checks if a line is potentially isolated (no markers, no context)
func isIsolatedLine(line string, frontLen int, previousLine string, tp *template.TemplateProcessor) bool {
	if line == "" || note.IsClozeFront(line) || note.IsBasicFront(line) ||
		previousLine != "" || frontLen > 0 {
		return false
	}

	// Check if it's a template line
	for templateName := range tp.Templates {
		if strings.HasPrefix(line, templateName+" ") {
			return false
		}
	}

	// Allow comma-separated lists
	return !strings.Contains(line, ",")
}

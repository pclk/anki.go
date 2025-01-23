package note

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/pclk/anki.go/converter/types"
)

func WriteCloze(writer *bufio.Writer, text, backExtra string, counter *int, currentSections []types.Section) {
	fmt.Printf("Writing CLOZE deletion - Text: %q\n", strings.TrimSpace(text))
	writer.WriteString("# Note\n")
	writer.WriteString("model: Cloze\n\n")
	writer.WriteString("## Text\n\n")

	// Add section metadata before the content
	if len(currentSections) > 0 {
		mainSectionWritten := false
		for i, section := range currentSections {
			if section.Level <= 2 && !mainSectionWritten {
				writer.WriteString("Section: " + section.Content + "\n")
				mainSectionWritten = true
			} else if section.Level > 2 && mainSectionWritten && i == len(currentSections)-1 {
				writer.WriteString("Sub-section: " + section.Content + "\n")
			}
		}
		writer.WriteString("\n")
	}

	// Counter for generating unique cloze numbers (c1, c2, etc.)
	clozeCounter := 0
	// Maps group numbers to cloze numbers for grouped deletions (e.g., "1.text" and "1.another" share same cloze number)
	groupMap := make(map[string]int)

	// processed accumulates the transformed text
	processed := ""
	// remaining holds the unprocessed portion of the text
	remaining := text

	// Keep processing while there are cloze deletions (text between hyphens) to handle
	for {
		// Find the start of a potential cloze deletion (first unescaped hyphen)
		start := -1
		for i := 0; i < len(remaining); i++ {
			if remaining[i] == '-' {
				// Check if hyphen is escaped
				if i > 0 && remaining[i-1] == '\\' {
					continue
				}
				start = i
				break
			}
		}
		if start == -1 { // No more unescaped hyphens found
			processed += remaining // Add remaining text and stop
			break
		}

		// Find the end of the cloze deletion (next unescaped hyphen)
		end := -1
		for i := start + 1; i < len(remaining); i++ {
			if remaining[i] == '-' {
				// Check if hyphen is escaped
				if i > 0 && remaining[i-1] == '\\' {
					continue
				}
				end = i
				break
			}
		}
		if end == -1 { // No matching end hyphen
			processed += remaining // Add remaining text and stop
			break
		}

		// Look for optional group numbers before cloze (format: "1.text" where 1 is the group)
		groupNum := ""
		// Get all text before the current cloze deletion
		beforeCloze := remaining[:start]
		// Extract the text between the hyphens that will become the cloze deletion
		clozeText := remaining[start+1 : end]

		fmt.Printf("\nCloze Processing State:\n")
		fmt.Printf("  Full remaining text: %q\n", remaining)
		fmt.Printf("  Current start position: %d\n", start)
		fmt.Printf("  Current end position: %d\n", end)
		fmt.Printf("  Text between hyphens: %q\n", clozeText)
		fmt.Printf("  beforeCloze: %q\n", beforeCloze)
		fmt.Printf("  processed length: %d\n", len(processed))
		fmt.Printf("  processed content: %q\n", processed)
		fmt.Printf("  remaining after this cloze will be: %q\n", remaining[end+1:])

		potentialNum := ""
		// Only process if there's text before the cloze to check for group numbers
		if len(beforeCloze) > 0 {
			// Look for the last dot before this cloze - group numbers are in format "1."
			lastDot := strings.LastIndex(beforeCloze, ".")

			// Ensure we found a dot and it's at the end of beforeCloze.
			if lastDot != -1 && lastDot == len(beforeCloze)-1 {
				fmt.Println("  dot is found at the end of beforeCloze.")
				fmt.Printf("  checking substring from %d to %d\n", lastDot-1, lastDot)
				// Extract the character before the dot and trim whitespace
				// For input like "1.text", this gets "1"
				potentialNum = strings.TrimSpace(beforeCloze[lastDot-1 : lastDot])
				fmt.Printf("  potentialNum: %q\n", potentialNum)
				// Use the number if it's not empty after trimming
				if potentialNum != "" {
					groupNum = potentialNum
					fmt.Printf("  found group number: %q\n", groupNum)
				}
			}
		}

		// Add all text before start hyphen to output
		processed += remaining[:start]

		if groupNum != "" {
			// Remove the group number and dot we just added
			processed = processed[:len(processed)-(len(potentialNum)+1)]
			fmt.Printf("  removed group number from processed. processed: %v", processed)

			// This cloze has a group number (e.g., "1.-cat- 1.-dog-")
			if num, exists := groupMap[groupNum]; exists {
				// If we've seen this group number before, use its saved cloze number
				clozeFormatted := fmt.Sprintf("{{c%d::%s}}", num, clozeText)
				processed += clozeFormatted
			} else {
				// If this is a new group, increment counter and save it
				clozeCounter++
				groupMap[groupNum] = clozeCounter
				clozeFormatted := fmt.Sprintf("{{c%d::%s}}", clozeCounter, clozeText)
				processed += clozeFormatted
			}
		} else {
			// No group number, just use next number
			clozeCounter++
			clozeFormatted := fmt.Sprintf("{{c%d::%s}}", clozeCounter, clozeText)
			processed += clozeFormatted
		}
		// Update remaining to continue after this cloze
		remaining = remaining[end+1:]
	}

	writer.WriteString(processed + "\n\n")

	writer.WriteString("## Back Extra\n\n")
	if backExtra != "" {
		writer.WriteString(backExtra + "\n\n")
	}
	writer.WriteString("\n")
}

func IsClozeFront(text string) bool {
	// Skip if it's a section header with ">"
	if strings.Contains(text, "Sub-section:") && strings.Contains(text, " > ") {
		return false
	}

	// Look for pairs of hyphens that aren't part of other constructs
	count := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '-' {
			// Skip if it's part of " > " separator
			if i > 0 && i < len(text)-1 && text[i-1] == ' ' && text[i+1] == ' ' {
				continue
			}
			count++
		}
	}
	return count >= 2 && count%2 == 0
}

package converter

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/pclk/anki.go/fileops"
)

var traceEnabled = true

func trace(format string, args ...interface{}) {
	if traceEnabled {
		fmt.Printf(format+"\n", args...)
	}
}

func ExtractDeckName(scanner *bufio.Scanner) string {
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# deck:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# deck:"))
		}
	}
	return ""
}

func ConvertToAnki(inputFile, outputFile string) (string, error) {
	trace("Starting conversion from %s to %s", inputFile, outputFile)
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

	if err := processFile(scanner, writer, inputFile); err != nil {
		return "", err
	}
	return deckName, nil
}

/*
Template Rules:

1. Template Definition:
   - Templates are defined at the start of the deck before any cards
   - Format: "templateName: template text"
   - Must contain at least one {} placeholder
   - Special syntax "-{,}-" indicates comma-separated cloze deletions
   - Example: "def: What is {}? | {}"
   - Example with cloze: "ex: {} examples of {} are -{,}-"

2. Template Usage Formats:
   a) Single-line with pipe:
      templateName param1 | param2
      Example: "def Algorithm | A step-by-step procedure"
      Example: "ex 3 | Binary Tree | BST, AVL Tree, Red-Black Tree"

   b) Multi-line with pipe:
      templateName param1 | param2
      param3
      Example:
      def Algorithm | A step-by-step procedure
      Example:
      ex 3 | Binary Tree
      BST, AVL Tree, Red-Black Tree

   c) Multi-line without pipe:
      templateName param1
      param2
      Example:
      def Algorithm
      A step-by-step procedure
      Example:
      ex 3
      Binary Tree
      BST, AVL Tree, Red-Black Tree

3. Special Syntax:
   "-{,}-" in template definition:
   - Indicates that parameter should be split by commas
   - Each item automatically wrapped in consecutive cloze deletions
   - Example input: "BST, AVL Tree, Red-Black Tree"
   - Becomes: "{{c1::BST}}, {{c2::AVL Tree}}, {{c3::Red-Black Tree}}"

4. Examples:
   Template: "ex: {} examples of {} are -{,}-"

   Usage: "ex 3 | Binary Tree | BST, AVL Tree, Red-Black Tree"
   Output: "3 examples of Binary Tree are {{c1::BST}}, {{c2::AVL Tree}}, {{c3::Red-Black Tree}}"

   Usage:
   ex 3 | Binary Tree
   BST, AVL Tree, Red-Black Tree
   Output: "3 examples of Binary Tree are {{c1::BST}}, {{c2::AVL Tree}}, {{c3::Red-Black Tree}}"
*/

// Template represents a card template definition
type Template struct {
	Name     string
	Content  string
	HasCloze bool
}

// TemplateProcessor handles template definitions and usage
type TemplateProcessor struct {
	templates map[string]*Template
}

func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		templates: make(map[string]*Template),
	}
}

// parseTemplate parses a template definition line
func (tp *TemplateProcessor) parseTemplate(line string) bool {
	trace("Attempting to parse template: %s", line)
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return false
	}

	name := strings.TrimSpace(parts[0])
	content := strings.TrimSpace(parts[1])

	// Check if it's a valid template (contains {} and starts with word characters)
	if !strings.Contains(content, "{}") {
		return false
	}

	// Store template without the ":" in the name
	tp.templates[name] = &Template{
		Name:     name,
		Content:  content,
		HasCloze: strings.Contains(content, "-{,}-"),
	}
	return true
}

// applyTemplate applies a template to given parameters
func (tp *TemplateProcessor) applyTemplate(templateName string, params []string) (string, error) {
	trace("Applying template:")
	trace("  Template name: %q", templateName)
	trace("  Parameters: %s", strings.Join(params, ", "))
	template, exists := tp.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	result := template.Content

	// Special handling for cloze template
	if template.HasCloze && len(params) > 0 {
		// Get the last parameter which should contain comma-separated items
		lastParam := params[len(params)-1]
		items := strings.Split(lastParam, ",")

		// Create cloze deletions
		var clozeItems []string
		for i, item := range items {
			clozeItems = append(clozeItems, fmt.Sprintf("{{c%d::%s}}", i+1, strings.TrimSpace(item)))
		}

		// Replace the cloze placeholder with formatted items
		result = strings.Replace(result, "-{,}-", strings.Join(clozeItems, ", "), 1)

		// Replace remaining placeholders with other params
		for _, param := range params[:len(params)-1] {
			result = strings.Replace(result, "{}", param, 1)
		}
	} else {
		// Normal template processing
		for _, param := range params {
			result = strings.Replace(result, "{}", param, 1)
		}
	}

	return result, nil
}

// hasTemplateUsage checks if a line uses any of the defined templates
func (tp *TemplateProcessor) hasTemplateUsage(line string) (string, bool) {
	trimmedLine := strings.TrimSpace(line)
	for templateName := range tp.templates {
		if strings.HasPrefix(trimmedLine, templateName+" ") {
			return templateName, true
		}
	}
	return "", false
}

// processTemplateOrLine handles both template processing and regular line processing
func processTemplateOrLine(line string, front, back *strings.Builder, writer *bufio.Writer, currentSection *string, scanner *bufio.Scanner, tp *TemplateProcessor) (bool, error) {
	trimmedLine := strings.TrimSpace(line)

	// Skip template definitions in second pass
	if strings.Contains(trimmedLine, ":") {
		if tp.parseTemplate(line) {
			return true, nil
		}
	}

	// Try to apply template
	templateName, templateFound := tp.hasTemplateUsage(trimmedLine)
	// Handle template usage
	trace("\nChecking line %q", trimmedLine)
	if templateFound {
		templateFound = true
		trace("  ✓ Template %q matches!", templateName)
		trace("  Collecting parameters:")
		trace("    Initial content: %q", trimmedLine)
		var params []string
		content := strings.TrimSpace(strings.TrimPrefix(trimmedLine, templateName))
		trace("    After template name removal: %q", content)

		// Split first line by pipe if it exists
		if strings.Contains(content, "|") {
			trace("    Found pipe separator")
			parts := strings.Split(content, "|")
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					params = append(params, trimmed)
					trace("    Parameter added from pipe: %q", trimmed)
				}
			}
		} else {
			// If no pipe, add entire content as first parameter
			if content != "" {
				params = append(params, content)
				trace("    Single parameter: %q", content)
			}
		}

		// Look ahead for additional content until blank line
		for scanner.Scan() {
			nextLine := strings.TrimSpace(scanner.Text())
			if nextLine == "" {
				break
			}
			params = append(params, nextLine)
			trace("    Additional line parameter: %q", nextLine)
		}

		// Apply template
		if result, err := tp.applyTemplate(templateName, params); err == nil {
			// Write out previous note if exists
			if front.Len() > 0 || back.Len() > 0 {
				if hasClozeMarkers(front.String()) {
					writeClozeDeletion(writer, front.String(), back.String(), nil)
				} else {
					// Verify basic notes end with a question mark
					trimmedFront := strings.TrimSpace(front.String())
					if !strings.HasSuffix(trimmedFront, "?") {
						fmt.Printf("\n⚠️  WARNING: Basic note front does not end with a question mark:\n%s\n", trimmedFront)
						fmt.Print("Continue processing? (y/N): ")
						var response string
						fmt.Scanln(&response)
						if response != "y" && response != "Y" {
							return false, fmt.Errorf("process stopped by user")
						}
					}
					writeBasicNote(writer, front.String(), back.String())
				}
				front.Reset()
				back.Reset()
			}

			// Write the processed template result directly
			if strings.Contains(result, "{{c") {
				writeClozeDeletion(writer, result, "", nil)
			} else if strings.Contains(result, "|") {
				parts := strings.Split(result, "|")
				writeBasicNote(writer, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			} else {
				writeBasicNote(writer, result, "")
			}
			return true, nil
		}
	}

	if !templateFound {
		trace("Template not found")
		return false, processLine(line, front, back, writer, currentSection, scanner, tp)
	}
	return true, nil
}

func processFile(scanner *bufio.Scanner, writer *bufio.Writer, inputFile string) error {
	trace("Processing file: %s", inputFile)
	var front, back strings.Builder
	var currentSection string
	foundDeck := false
	previousLine = ""
	lineNumber := 0
	tp := NewTemplateProcessor()

	// Skip everything until we find the deck line
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.HasPrefix(strings.ToLower(line), "# deck:") {
			foundDeck = true
			break
		}
	}

	if !foundDeck {
		return fmt.Errorf("no deck specification found")
	}

	trace("Starting template collection phase")
	// Collect templates first
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines
		if trimmedLine == "" {
			trace("\nSkipping empty line during template collection phase")
			continue
		}

		// Stop collecting templates when we hit a non-template line
		if !strings.Contains(trimmedLine, ":") || !tp.parseTemplate(line) {
			// Debug print templates
			fmt.Println("\nStored Templates:")
			for name, template := range tp.templates {
				fmt.Printf("Name: %s\nContent: %s\nHas Cloze: %v\n\n",
					name, template.Content, template.HasCloze)
			}

			// Don't process the line here, just break the template collection phase
			break
		}
	}

	// Process remaining lines, starting with the last line we read
	line := scanner.Text()
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine != "" {
		_, err := processTemplateOrLine(line, &front, &back, writer, &currentSection, scanner, tp)
		if err != nil {
			return err
		}
	}

	// var lastLine string
	for scanner.Scan() {
		line := scanner.Text()
		// lastLine = line // Store the last line
		trimmedLine := strings.TrimSpace(line)

		// Use empty lines as card separators
		if trimmedLine == "" {
			trace("\nEncountered empty line - checking for card to write")
			if front.Len() > 0 || back.Len() > 0 {
				trace("  Writing card with:")
				trace("    Front buffer length: %d", front.Len())
				if hasClozeMarkers(front.String()) {
					writeClozeDeletion(writer, front.String(), back.String(), nil)
				} else {
					writeBasicNote(writer, front.String(), back.String())
				}
				front.Reset()
				back.Reset()
			} else {
				trace("  No content to write - skipping")
			}
			previousLine = line
			continue
		}

		if _, err := processTemplateOrLine(line, &front, &back, writer, &currentSection, scanner, tp); err != nil {
			return err
		}
		previousLine = line
	}

	// // Process the final line if it's not empty
	// if strings.TrimSpace(lastLine) != "" {
	// 	trace("\nProcessing final line")
	//
	// 	// Check if the final line is isolated
	// 	trimmedLine := strings.TrimSpace(lastLine)
	// 	if trimmedLine != "" && !hasClozeMarkers(lastLine) && !strings.HasSuffix(lastLine, "?") {
	// 		// Skip warning for template lines and comma-separated lists
	// 		isTemplateLine := false
	// 		for templateName := range tp.templates {
	// 			if strings.HasPrefix(trimmedLine, templateName+" ") {
	// 				isTemplateLine = true
	// 				break
	// 			}
	// 		}
	// 		if !isTemplateLine && !strings.Contains(trimmedLine, ",") {
	// 			fmt.Printf("\n⚠️  WARNING: Isolated line at end of file without cloze markers or question mark:\n%s\n", lastLine)
	// 			fmt.Print("Continue processing? (y/N): ")
	// 			var response string
	// 			fmt.Scanln(&response)
	// 			if response != "y" && response != "Y" {
	// 				return fmt.Errorf("process stopped by user")
	// 			}
	// 			writer.WriteString("\n")
	// 		}
	// 	}
	//
	// 	// Only process if front and back are empty (not already processed)
	// 	if front.Len() == 0 && back.Len() == 0 {
	// 		if _, err := processTemplateOrLine(lastLine, &front, &back, writer, &currentSection, scanner, tp); err != nil {
	// 			return err
	// 		}
	// 	}

	// Add a newline at the end of the file
	// writer.WriteString("\n")
	// }

	// Write any remaining content
	if front.Len() > 0 || back.Len() > 0 {
		trace("\nWriting final card:")
		trace("  Front buffer length: %d", front.Len())
		trace("  Back buffer length: %d", back.Len())
		if hasClozeMarkers(front.String()) {
			writeClozeDeletion(writer, front.String(), back.String(), nil)
		} else {
			// Verify basic notes end with a question mark
			trimmedFront := strings.TrimSpace(front.String())
			if !strings.HasSuffix(trimmedFront, "?") {
				fmt.Printf("\n⚠️  WARNING: Basic note front does not end with a question mark:\n%s\n", trimmedFront)
				fmt.Print("Continue processing? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					return fmt.Errorf("process stopped by user")
				}
			}
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

func processLine(line string, front, back *strings.Builder, writer *bufio.Writer, currentSection *string, scanner *bufio.Scanner, tp *TemplateProcessor) error {
	trace("Current state:")
	trace("  Front buffer length: %d", front.Len())
	trace("  Front buffer content: %q", front.String())
	trace("  Back buffer length: %d", back.Len())
	trace("  Back buffer content: %q", back.String())
	trace("Checking for section header...")
	if strings.HasPrefix(line, "#") {
		trace("  Found section header")
		// Write out any existing note before changing sections
		if front.Len() > 0 || back.Len() > 0 {
			if hasClozeMarkers(front.String()) {
				writeClozeDeletion(writer, front.String(), back.String(), nil)
			} else {
				writeBasicNote(writer, front.String(), back.String())
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
				trace("  Section level: %d", level)
				// Clear all sections if we're at level 1 or 2 or if section is titled "Clear section"
				currentContent := strings.TrimSpace(line[i:])
				if level <= 2 || strings.TrimSpace(currentContent) == "Clear section" {
					trace("  Clearing all sections (level <= 2 or Clear section)")
					currentSections = nil
				} else {
					trace("  Removing sections at or below level %d", level)
					// Remove sections at or below this level
					for len(currentSections) > 0 && currentSections[len(currentSections)-1].Level >= level {
						trace("    Removing section: %s (level %d)",
							currentSections[len(currentSections)-1].Content,
							currentSections[len(currentSections)-1].Level)
						currentSections = currentSections[:len(currentSections)-1]
					}
				}
				// Add new section
				sectionContent := strings.TrimSpace(line[i:])
				// Skip adding "Clear section" to the sections list
				if sectionContent != "Clear section" {
					trace("  Adding new section: %s (level %d)", sectionContent, level)
					currentSections = append(currentSections, Section{
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
	if (strings.HasPrefix(line, "#") || hasBasicQuestionmark(line)) && (front.Len() > 0 || back.Len() > 0) {
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
		trace("  Processing back extra for cloze")
		if back.Len() > 0 {
			back.WriteString("\n")
		}
		back.WriteString(strings.TrimSpace(strings.TrimPrefix(line, ">")))

	case hasClozeMarkers(line):
		trace("  Found cloze markers in line")
		if front.Len() > 0 || back.Len() > 0 {
			if hasClozeMarkers(front.String()) {
				writeClozeDeletion(writer, front.String(), back.String(), nil)
			} else {
				writeBasicNote(writer, front.String(), back.String())
			}
			front.Reset()
			back.Reset()
		}

		// Start new cloze deletion
		front.WriteString(line)

	case hasBasicQuestionmark(line):
		trace("  Found question mark suffix - starting new basic card")
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
			mainSectionWritten := false
			for i, section := range currentSections {
				if section.Level <= 2 && !mainSectionWritten {
					front.WriteString("Section: " + section.Content + "\n\n")
					mainSectionWritten = true
				} else if section.Level > 2 && mainSectionWritten {
					if i == len(currentSections)-1 {
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
		trace("  Processing default case")
		trimmedLine := strings.TrimSpace(line)

		// Handle basic card answer
		if strings.HasSuffix(strings.TrimSpace(front.String()), "?") {
			trace("    Adding line as answer to basic card")
			if back.Len() > 0 {
				back.WriteString("\n")
			}
			back.WriteString(line)
			return nil
		}

		// Check for isolated lines
		if isIsolatedLine(trimmedLine, front.Len(), previousLine, tp) {
			trace("    Found potential isolated line")
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
				fmt.Printf("\n⚠️  WARNING: Isolated line found:\n%s\n", line)
				fmt.Print("Continue processing? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					return fmt.Errorf("process stopped by user")
				}
				return processLine(nextLine, front, back, writer, currentSection, scanner, tp)
			}

			// Process the next line since this wasn't actually isolated
			return processLine(nextLine, front, back, writer, currentSection, scanner, tp)
		}

		// Add line to front of card
		trace("    Adding line to front of card")
		if front.Len() > 0 {
			front.WriteString("\n")
		}
		front.WriteString(line)
	}
	return nil
}

func writeBasicNote(writer *bufio.Writer, front, back string) {
	trace("Writing BASIC note:")
	trace("  Front: %q", front)
	trace("  Back: %q", back)
	writer.WriteString("# Note\n\n")
	writer.WriteString("## Front\n\n")
	writer.WriteString(strings.TrimSpace(front))
	writer.WriteString("\n\n")
	writer.WriteString("## Back\n\n")
	writer.WriteString(strings.TrimSpace(back))
	writer.WriteString("\n\n")
}

func hasBasicQuestionmark(text string) bool {
	trimmedText := strings.TrimSpace(text)
	return strings.HasSuffix(trimmedText, "?")
}

// isIsolatedLine checks if a line is potentially isolated (no markers, no context)
func isIsolatedLine(line string, frontLen int, previousLine string, tp *TemplateProcessor) bool {
	if line == "" || hasClozeMarkers(line) || hasBasicQuestionmark(line) ||
		previousLine != "" || frontLen > 0 {
		return false
	}

	// Check if it's a template line
	for templateName := range tp.templates {
		if strings.HasPrefix(line, templateName+" ") {
			return false
		}
	}

	// Allow comma-separated lists
	return !strings.Contains(line, ",")
}

func hasClozeMarkers(text string) bool {
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

func writeClozeDeletion(writer *bufio.Writer, text, backExtra string, counter *int) {
	trace("Writing CLOZE deletion:")
	trace("  Text: %q", text)
	trace("  Back Extra: %q", backExtra)
	writer.WriteString("# Note\n")
	writer.WriteString("model: Cloze\n\n")
	writer.WriteString("## Text\n\n")
	// Only add section headers if they're not already in the text
	if len(currentSections) > 0 && !strings.HasPrefix(strings.TrimSpace(text), "Section:") {
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

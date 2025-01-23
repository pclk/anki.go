package converter

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/pclk/anki.go/converter/note"
	"github.com/pclk/anki.go/converter/processor"
	"github.com/pclk/anki.go/converter/template"
	"github.com/pclk/anki.go/converter/types"
	"github.com/pclk/anki.go/fileops"
)

var nextLine string

// returns deckName, nextLine
func ExtractDeckName(scanner *bufio.Scanner) (string, string) {
	// by default split method, scanner is using ScanLines
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# deck:") {
			deckName := strings.TrimSpace(strings.TrimPrefix(line, "# deck:"))
			// Read the next line to position scanner after deck name
			if scanner.Scan() {
				return deckName, scanner.Text()
			}
			return deckName, ""
		}
	}
	return "", ""
}

func ConvertToAnki(inputFile, outputFile string) (string, error) {
	fmt.Printf("Starting conversion: %s -> %s\n", inputFile, outputFile)
	scanner, inputCleanup, err := fileops.OpenInputFile(inputFile)
	if err != nil {
		return "", err
	}
	defer inputCleanup()

	// Read deck name and get the next line
	var deckName string
	deckName, nextLine = ExtractDeckName(scanner)
	if deckName == "" {
		return "", fmt.Errorf(
			"Deck name cannot be found. Please include a `# deck: <deckName>` at the start of your cards.",
		)
	}

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

func processFile(scanner *bufio.Scanner, writer *bufio.Writer, inputFile string) error {
	fmt.Printf("Processing file: %s\n", inputFile)
	var front, back strings.Builder
	var currentSection string
	var currentSections []types.Section
	tp := template.NewTemplateProcessor()

	fmt.Printf("Starting template collection phase\n")

	// Process any templates starting with the line after deck name
	// We don't need to check for error because we're not breaking out of any loop.
	_ = tp.CollectTemplates(nextLine)

	// Continue collecting templates from scanner
	for scanner.Scan() {
		line := scanner.Text()
		if err := tp.CollectTemplates(line); err != nil {
			break // Non-template or empty line encountered
		}
	}

	// Log collected templates
	fmt.Printf("Template collection complete\n")
	for name, template := range tp.Templates {
		fmt.Printf("Template: %s, Content: %s, Has Cloze: %v\n",
			name, template.Content, template.HasCloze)
	}

	// var lastLine string
	for scanner.Scan() {
		line := scanner.Text()
		// lastLine = line // Store the last line
		trimmedLine := strings.TrimSpace(line)

		// Use empty lines as card separators
		if trimmedLine == "" {
			fmt.Printf("\nEncountered empty line - checking for card to write\n")
			if front.Len() > 0 || back.Len() > 0 {
				fmt.Printf("  Writing card with:\n")
				fmt.Printf("    Front buffer length: %d\n", front.Len())
				if note.IsClozeFront(front.String()) {
					note.WriteCloze(writer, front.String(), back.String(), nil, currentSections)
				} else {
					note.WriteBasic(writer, front.String(), back.String())
				}
				front.Reset()
				back.Reset()
			} else {
				fmt.Printf("  No content to write - skipping\n")
			}

			continue
		}

		if _, err := processTemplateOrLine(line, &front, &back, writer, &currentSection, currentSections, scanner, tp); err != nil {
			return err
		}
	}

	// Write any remaining content
	if front.Len() > 0 || back.Len() > 0 {
		fmt.Printf("\nWriting final card:\n")
		fmt.Printf("  Front buffer length: %d\n", front.Len())
		fmt.Printf("  Back buffer length: %d\n", back.Len())
		if note.IsClozeFront(front.String()) {
			note.WriteCloze(writer, front.String(), back.String(), nil, currentSections)
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
			note.WriteBasic(writer, front.String(), back.String())
		}
	}

	return writer.Flush()
}

// processTemplateOrLine handles both template processing and regular line processing
func processTemplateOrLine(line string, front, back *strings.Builder, writer *bufio.Writer, currentSection *string, currentSections []types.Section, scanner *bufio.Scanner, tp *template.TemplateProcessor) (bool, error) {
	trimmedLine := strings.TrimSpace(line)

	// Skip template definitions in second pass
	if strings.Contains(trimmedLine, ":") {
		if tp.ParseTemplate(line) {
			return true, nil
		}
	}

	// Try to apply template
	templateName, templateFound := tp.MatchTemplate(trimmedLine)
	// Handle template usage
	fmt.Printf("\nChecking line %q\n", trimmedLine)
	if templateFound {
		templateFound = true
		fmt.Printf("  ✓ Template %q matches!\n", templateName)
		fmt.Printf("  Collecting parameters:\n")
		fmt.Printf("    Initial content: %q\n", trimmedLine)
		var params []string
		content := strings.TrimSpace(strings.TrimPrefix(trimmedLine, templateName))
		fmt.Printf("    After template name removal: %q\n", content)

		// Split first line by pipe if it exists
		if strings.Contains(content, "|") {
			fmt.Printf("    Found pipe separator\n")
			parts := strings.Split(content, "|")
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					params = append(params, trimmed)
					fmt.Printf("    Parameter added from pipe: %q\n", trimmed)
				}
			}
		} else {
			// If no pipe, add entire content as first parameter
			if content != "" {
				params = append(params, content)
				fmt.Printf("    Single parameter: %q\n", content)
			}
		}

		// Look ahead for additional content until blank line
		for scanner.Scan() {
			nextLine := strings.TrimSpace(scanner.Text())
			if nextLine == "" {
				break
			}
			params = append(params, nextLine)
			fmt.Printf("    Additional line parameter: %q\n", nextLine)
		}

		// Apply template
		if result, err := tp.ApplyTemplate(templateName, params); err == nil {
			// Write out previous note if exists
			if front.Len() > 0 || back.Len() > 0 {
				if note.IsClozeFront(front.String()) {
					note.WriteCloze(writer, front.String(), back.String(), nil, currentSections)
				} else {
					// Verify basic notes end with a question mark
					trimmedFront := strings.TrimSpace(front.String())
					if !strings.HasSuffix(trimmedFront, "?") {
						fmt.Printf("WARNING: Basic note front does not end with a question mark: %s\n", trimmedFront)
						fmt.Print("Continue processing? (y/N): ")
						var response string
						fmt.Scanln(&response)
						if response != "y" && response != "Y" {
							return false, fmt.Errorf("process stopped by user")
						}
					}
					note.WriteBasic(writer, front.String(), back.String())
				}
				front.Reset()
				back.Reset()
			}

			// Write the processed template result directly
			if strings.Contains(result, "{{c") {
				note.WriteCloze(writer, result, "", nil, currentSections)
			} else if strings.Contains(result, "|") {
				parts := strings.Split(result, "|")
				note.WriteBasic(writer, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			} else {
				note.WriteBasic(writer, result, "")
			}
			return true, nil
		}
	}

	if !templateFound {
		fmt.Printf("Template not found\n")
		return false, processor.ProcessLine(line, front, back, writer, currentSection, currentSections, scanner, tp)
	}
	return true, nil
}

// isIsolatedLine checks if a line is potentially isolated (no markers, no context)
func isIsolatedLine(line string, frontLen int, tp *template.TemplateProcessor) bool {
	if line == "" || note.IsClozeFront(line) || note.IsBasicFront(line) ||
		frontLen > 0 {
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

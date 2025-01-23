package template

import (
	"fmt"
	"strings"
)

// Template represents a card template definition
type Template struct {
	Name     string
	Content  string
	HasCloze bool
}

// TemplateProcessor handles template definitions and usage
type TemplateProcessor struct {
	Templates map[string]*Template
}

func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		Templates: make(map[string]*Template),
	}
}

// parseTemplate parses a template definition line
func (tp *TemplateProcessor) ParseTemplate(line string) bool {
	fmt.Printf("Attempting to parse template: %s\n", line)
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
	tp.Templates[name] = &Template{
		Name:     name,
		Content:  content,
		HasCloze: strings.Contains(content, "-{,}-"),
	}
	return true
}

// applyTemplate applies a template to given parameters
func (tp *TemplateProcessor) ApplyTemplate(templateName string, params []string) (string, error) {
	fmt.Printf("Applying template:\n")
	fmt.Printf("  Template name: %q\n", templateName)
	fmt.Printf("  Parameters: %s\n", strings.Join(params, ", "))
	template, exists := tp.Templates[templateName]
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
func (tp *TemplateProcessor) MatchTemplate(line string) (string, bool) {
	trimmedLine := strings.TrimSpace(line)
	for templateName := range tp.Templates {
		if strings.HasPrefix(trimmedLine, templateName+" ") {
			return templateName, true
		}
	}
	return "", false
}

// collectTemplates processes a single line for template collection.
// Returns error when template collection should stop (empty line or non-template line).
func (tp *TemplateProcessor) CollectTemplates(line string) error {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		fmt.Printf("\nEmpty line encountered - ending template collection phase\n")
		return fmt.Errorf("empty line")
	}

	if tp.ParseTemplate(line) {
		fmt.Printf("Successfully parsed template from line: %s\n", line)
		return nil
	}

	fmt.Printf("Line is not a valid template definition: %s\n", line)
	return fmt.Errorf("non-template line")
}

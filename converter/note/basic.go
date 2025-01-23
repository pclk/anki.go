package note

import (
	"bufio"
	"fmt"
	"strings"
)

func WriteBasic(writer *bufio.Writer, front, back string) {
	fmt.Printf("Writing BASIC note - Front: %q\n", strings.TrimSpace(front))
	writer.WriteString("# Note\n\n")
	writer.WriteString("## Front\n\n")
	writer.WriteString(strings.TrimSpace(front))
	writer.WriteString("\n\n")
	writer.WriteString("## Back\n\n")
	writer.WriteString(strings.TrimSpace(back))
	writer.WriteString("\n\n")
}

func IsBasicFront(text string) bool {
	trimmedText := strings.TrimSpace(text)
	return strings.HasSuffix(trimmedText, "?")
}

package anthropic

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func ContinueConversation(message, pdfPath, apiKey, model, conversationFile string) error {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
	}

	// Create default conversation file if not specified
	if conversationFile == "" {
		conversationFile = filepath.Join("conversations", fmt.Sprintf("conversation_%d.json", time.Now().Unix()))
	}

	// Load or create conversation
	conv, err := LoadConversation(conversationFile)
	if err != nil {
		return fmt.Errorf("error loading conversation: %v", err)
	}

	// Update model if specified
	if model != "" {
		conv.Model = model
	}

	client := NewClient(apiKey, conv.Model)

	// Send message with history
	response, err := client.SendMessage(message, pdfPath, conv.Messages)
	if err != nil {
		return fmt.Errorf("error getting response from Claude: %v", err)
	}

	if pdfPath != "" {
		pdfInfo := DocumentInfo{
			Name:      filepath.Base(pdfPath),
			Type:      "application/pdf",
			Timestamp: time.Now(),
		}

		conv.Messages = append(conv.Messages,
			ConversationMessage{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now(),
				Documents: []DocumentInfo{pdfInfo},
			},
			ConversationMessage{
				Role:      "assistant",
				Content:   response,
				Timestamp: time.Now(),
			},
		)
	} else {
		conv.Messages = append(conv.Messages,
			ConversationMessage{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now(),
			},
			ConversationMessage{
				Role:      "assistant",
				Content:   response,
				Timestamp: time.Now(),
			},
		)
	}
	// Save updated conversation
	if err := SaveConversationFile(conv, conversationFile); err != nil {
		return fmt.Errorf("error saving conversation: %v", err)
	}

	fmt.Printf("\nClaude's response:\n%s\n", response)
	fmt.Printf("\nConversation saved to: %s\n", conversationFile)

	return nil
}

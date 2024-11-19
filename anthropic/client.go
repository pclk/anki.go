package anthropic

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func NewClient(apiKey string, model string) *Client {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &Client{
		ApiKey: apiKey,
		Model:  model,
	}
}

func (c *Client) SendMessage(message, pdfPath string, history []ConversationMessage) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	var jsonData []byte
	var err error
	if pdfPath != "" {
		pdfData, err := os.ReadFile(pdfPath)
		if err != nil {
			return "", fmt.Errorf("error reading PDF file: %v", err)
		}

		encodedPDF := base64.StdEncoding.EncodeToString(pdfData)

		// Create message content with document
		content := []ContentBlock{
			{
				Type: "document",
				Source: &DocumentSource{
					Type:      "base64",
					MediaType: "application/pdf",
					Data:      encodedPDF,
				},
			},
			{
				Type: "text",
				Text: message,
			},
		}
		// Convert history to message format
		messages := make([]MessageContent, 0, len(history)+1)
		for _, msg := range history {
			msgContent := []ContentBlock{{
				Type: "text",
				Text: msg.Content,
			}}
			messages = append(messages, MessageContent{
				Role:    msg.Role,
				Content: msgContent,
			})
		}
		messages = append(messages, MessageContent{
			Role:    "user",
			Content: content,
		})
		request := AnthropicRequestWithDoc{
			Model:     c.Model,
			MaxTokens: 8192,
			Messages:  messages,
		}
		jsonData, err = json.Marshal(request)
		if err != nil {
			return "", fmt.Errorf("error marshaling request: %v", err)
		}
	} else {
		// Convert conversation history to API message format
		messages := make([]Message, 0, len(history)+1)
		for _, msg := range history {
			messages = append(messages, Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Add new message
		messages = append(messages, Message{
			Role:    "user",
			Content: message,
		})

		request := AnthropicRequest{
			Model:     c.Model,
			MaxTokens: 8192,
			Messages:  messages,
		}

		jsonData, err = json.Marshal(request)
		if err != nil {
			return "", fmt.Errorf("error marshaling request: %v", err)
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.ApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	if pdfPath != "" {
		req.Header.Set("anthropic-beta", "pdfs-2024-09-25")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Content[0].Text, nil
}

func LoadConversation(filename string) (*ConversationFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new conversation file
			return &ConversationFile{
				ID:       fmt.Sprintf("conv_%d", time.Now().Unix()),
				Created:  time.Now(),
				Updated:  time.Now(),
				Messages: make([]ConversationMessage, 0),
				Metadata: make(map[string]string),
			}, nil
		}
		return nil, fmt.Errorf("error reading conversation file: %v", err)
	}

	var conv ConversationFile
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("error parsing conversation file: %v", err)
	}

	return &conv, nil
}

func SaveConversationFile(conv *ConversationFile, filename string) error {
	// Create conversations directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	// Update timestamp
	conv.Updated = time.Now()

	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling conversation: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("error writing conversation file: %v", err)
	}

	return nil
}

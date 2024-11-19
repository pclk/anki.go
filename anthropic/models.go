package anthropic

import "time"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}
type AnthropicRequestWithDoc struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []MessageContent `json:"messages"`
}

type ContentItem struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type MessageContent struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type AnthropicResponse struct {
	Content []ContentItem `json:"content"`
	Role    string        `json:"role"`
}

type ConversationMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Timestamp time.Time      `json:"timestamp"`
	Documents []DocumentInfo `json:"documents,omitempty"`
}

type ConversationFile struct {
	ID       string                `json:"id"`
	Model    string                `json:"model"`
	Created  time.Time             `json:"created"`
	Updated  time.Time             `json:"updated"`
	Messages []ConversationMessage `json:"messages"`
	Metadata map[string]string     `json:"metadata,omitempty"`
}

type DocumentInfo struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

type DocumentSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ContentBlock struct {
	Type   string          `json:"type"`
	Text   string          `json:"text,omitempty"`
	Source *DocumentSource `json:"source,omitempty"`
}

type Client struct {
	ApiKey string
	Model  string
}

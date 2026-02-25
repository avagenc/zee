package chat

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model wraps the OpenAI client configured for Groq.
// It implements the google.golang.org/adk/model.LLM interface.
type Model struct {
	client *openai.Client
	name   string
}

func NewModel(client *openai.Client, name string) *Model {
	return &Model{
		client: client,
		name:   name,
	}
}

// Name returns the name of the model.
func (m *Model) Name() string {
	return m.name
}

// GenerateContent generates content from the model.
// ADK uses iter.Seq2 to handle streaming. We currently support only non-streaming for simplicity in this adapter.
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK's genai.Content to OpenAI messages
		var messages []openai.ChatCompletionMessage
		for _, content := range req.Contents {
			role := content.Role
			if role == "" {
				role = openai.ChatMessageRoleUser
			} else if role == "model" {
				role = openai.ChatMessageRoleAssistant
			}

			// We only handle simple text parts for now
			var textContent string
			for _, part := range content.Parts {
				if part.Text != "" {
					textContent += part.Text
				}
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:    role,
				Content: textContent,
			})
		}

		if req.Config != nil && req.Config.SystemInstruction != nil {
			var systemText string
			for _, part := range req.Config.SystemInstruction.Parts {
				if part.Text != "" {
					systemText += part.Text
				}
			}
			if systemText != "" {
				// Prepend system message
				messages = append([]openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: systemText,
					},
				}, messages...)
			}
		}

		log.Printf("[Groq Adapter] Sending request to %s with %d messages", m.name, len(messages))

		groqReq := openai.ChatCompletionRequest{
			Model:    m.Name(),
			Messages: messages,
		}

		if req.Config != nil && req.Config.Temperature != nil {
			groqReq.Temperature = float32(*req.Config.Temperature)
		}

		resp, err := m.client.CreateChatCompletion(ctx, groqReq)
		if err != nil {
			yield(nil, fmt.Errorf("groq completion error: %w", err))
			return
		}

		if len(resp.Choices) == 0 {
			yield(nil, errors.New("groq returned no choices"))
			return
		}

		// Convert back to ADK model response
		adkResponse := &model.LLMResponse{
			Content: &genai.Content{
				Role: "model",
				Parts: []*genai.Part{
					{Text: resp.Choices[0].Message.Content},
				},
			},
			TurnComplete: true,
			FinishReason: genai.FinishReasonStop,
		}

		yield(adkResponse, nil)
	}
}

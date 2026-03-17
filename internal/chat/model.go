package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

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

func (m *Model) Name() string {
	return m.name
}

func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		var messages []openai.ChatCompletionMessage
		for _, content := range req.Contents {
			role := content.Role
			if role == "" {
				role = openai.ChatMessageRoleUser
			} else if role == "model" {
				role = openai.ChatMessageRoleAssistant
			}

			var textContent string
			var toolCalls []openai.ToolCall
			var toolCallID string

			for _, part := range content.Parts {
				if part.Text != "" {
					textContent += part.Text
				}
				if part.FunctionCall != nil {
					argsBytes, _ := json.Marshal(part.FunctionCall.Args)
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   part.FunctionCall.Name + "_id",
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsBytes),
						},
					})
				}
				if part.FunctionResponse != nil {
					role = openai.ChatMessageRoleTool
					toolCallID = part.FunctionResponse.Name + "_id"
					respBytes, _ := json.Marshal(part.FunctionResponse.Response)
					textContent = string(respBytes)
				}
			}

			msg := openai.ChatCompletionMessage{
				Role:       role,
				Content:    textContent,
				ToolCalls:  toolCalls,
				ToolCallID: toolCallID,
			}
			messages = append(messages, msg)
		}

		if req.Config != nil && req.Config.SystemInstruction != nil {
			var systemText string
			for _, part := range req.Config.SystemInstruction.Parts {
				if part.Text != "" {
					systemText += part.Text
				}
			}
			if systemText != "" {
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

		if req.Config != nil && len(req.Config.Tools) > 0 {
			var mapSchema func(s *genai.Schema) map[string]any
			mapSchema = func(s *genai.Schema) map[string]any {
				if s == nil {
					return nil
				}
				m := make(map[string]any)
				if s.Type != "" {
					switch s.Type {
					case genai.TypeString:
						m["type"] = "string"
					case genai.TypeNumber:
						m["type"] = "number"
					case genai.TypeInteger:
						m["type"] = "integer"
					case genai.TypeBoolean:
						m["type"] = "boolean"
					case genai.TypeArray:
						m["type"] = "array"
					case genai.TypeObject:
						m["type"] = "object"
					default:
						m["type"] = "string"
					}
				}
				if s.Format != "" {
					m["format"] = s.Format
				}
				if s.Description != "" {
					m["description"] = s.Description
				}
				if len(s.Enum) > 0 {
					m["enum"] = s.Enum
				}
				if len(s.Required) > 0 {
					m["required"] = s.Required
				}
				if s.Items != nil {
					m["items"] = mapSchema(s.Items)
				}
				if len(s.Properties) > 0 {
					props := make(map[string]any)
					for k, v := range s.Properties {
						props[k] = mapSchema(v)
					}
					m["properties"] = props
				}
				return m
			}

			for _, t := range req.Config.Tools {
				for _, fd := range t.FunctionDeclarations {
					schemaMap := mapSchema(fd.Parameters)
					schemaBytes, _ := json.Marshal(schemaMap)
					groqReq.Tools = append(groqReq.Tools, openai.Tool{
						Type: openai.ToolTypeFunction,
						Function: &openai.FunctionDefinition{
							Name:        fd.Name,
							Description: fd.Description,
							Parameters:  json.RawMessage(schemaBytes),
						},
					})
				}
			}
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

		choice := resp.Choices[0]

		parts := []*genai.Part{}
		if choice.Message.Content != "" {
			parts = append(parts, &genai.Part{Text: choice.Message.Content})
		}

		for _, tc := range choice.Message.ToolCalls {
			if tc.Type == openai.ToolTypeFunction {
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					args = make(map[string]any)
				}
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}
		}

		adkResponse := &model.LLMResponse{
			Content: &genai.Content{
				Role:  "model",
				Parts: parts,
			},
			TurnComplete: true,
			FinishReason: genai.FinishReasonStop,
		}

		yield(adkResponse, nil)
	}
}

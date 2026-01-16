// Package openai provides an OpenAI API implementation of the Provider interface.
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lex00/wetwire-core-go/providers"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// DefaultModel is the default model used by the OpenAI provider.
const DefaultModel = string(openai.ChatModelGPT4o)

// Provider implements the providers.Provider interface using the OpenAI API.
type Provider struct {
	client *openai.Client
}

// Config contains configuration for the OpenAI provider.
type Config struct {
	// APIKey for OpenAI (defaults to OPENAI_API_KEY env var)
	APIKey string
}

// New creates a new OpenAI provider.
func New(config Config) (*Provider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &Provider{
		client: &client,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai"
}

// CreateMessage sends a message request and returns the complete response.
func (p *Provider) CreateMessage(ctx context.Context, req providers.MessageRequest) (*providers.MessageResponse, error) {
	params := p.buildParams(req)

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	return p.convertResponse(resp), nil
}

// StreamMessage sends a message request and streams the response via the handler.
func (p *Provider) StreamMessage(ctx context.Context, req providers.MessageRequest, handler providers.StreamHandler) (*providers.MessageResponse, error) {
	params := p.buildParams(req)

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	var fullText strings.Builder
	var toolCalls []openai.ChatCompletionMessageToolCall
	var finishReason string

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// Handle text delta
			if choice.Delta.Content != "" {
				handler(choice.Delta.Content)
				fullText.WriteString(choice.Delta.Content)
			}

			// Handle tool calls
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					// Extend toolCalls array if needed
					for len(toolCalls) <= int(tc.Index) {
						toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCall{})
					}

					// Accumulate tool call data
					existing := &toolCalls[tc.Index]
					if tc.ID != "" {
						existing.ID = tc.ID
					}
					// Skip Type assignment - it's a constant in the response
					if tc.Function.Name != "" {
						existing.Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						existing.Function.Arguments += tc.Function.Arguments
					}
				}
			}

			// Capture finish reason
			if choice.FinishReason != "" {
				finishReason = string(choice.FinishReason)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("streaming failed: %w", err)
	}

	// Build response
	response := &providers.MessageResponse{
		StopReason: convertFinishReason(finishReason),
	}

	// Add text content if present
	if fullText.Len() > 0 {
		response.Content = append(response.Content, providers.ContentBlock{
			Type: "text",
			Text: fullText.String(),
		})
	}

	// Add tool calls if present
	for _, tc := range toolCalls {
		if tc.ID != "" {
			response.Content = append(response.Content, providers.ContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			})
		}
	}

	return response, nil
}

// buildParams converts a MessageRequest to OpenAI API parameters.
func (p *Provider) buildParams(req providers.MessageRequest) openai.ChatCompletionNewParams {
	model := req.Model
	if model == "" {
		model = DefaultModel
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := openai.ChatCompletionNewParams{
		Model:     openai.ChatModel(model),
		MaxTokens: openai.Int(int64(maxTokens)),
	}

	// Convert messages
	var messages []openai.ChatCompletionMessageParamUnion

	// Add system message if present
	if req.System != "" {
		messages = append(messages, openai.SystemMessage(req.System))
	}

	// Convert conversation messages
	for _, msg := range req.Messages {
		messages = append(messages, p.convertMessage(msg))
	}

	params.Messages = messages

	// Convert tools if present
	if len(req.Tools) > 0 {
		params.Tools = p.convertTools(req.Tools)
	}

	return params
}

// convertMessage converts a provider message to an OpenAI message param.
func (p *Provider) convertMessage(msg providers.Message) openai.ChatCompletionMessageParamUnion {
	if msg.Role == "user" {
		// User message - extract text
		var textParts []string
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				textParts = append(textParts, block.Text)
			case "tool_result":
				// OpenAI uses tool messages for results
				// For now, we'll include as text
				textParts = append(textParts, fmt.Sprintf("[Tool %s result: %s]", block.ToolUseID, block.Content))
			}
		}
		return openai.UserMessage(strings.Join(textParts, "\n"))
	}

	// Assistant message
	var text string
	var toolCalls []openai.ChatCompletionMessageToolCallParam

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			text = block.Text
		case "tool_use":
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
				ID:   block.ID,
				Type: "function",
				Function: openai.ChatCompletionMessageToolCallFunctionParam{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}

	if len(toolCalls) > 0 {
		msgParam := openai.ChatCompletionAssistantMessageParam{
			Role:      "assistant",
			ToolCalls: toolCalls,
		}
		if text != "" {
			msgParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
				OfString: openai.String(text),
			}
		}
		return openai.ChatCompletionMessageParamUnion{
			OfAssistant: &msgParam,
		}
	}

	return openai.AssistantMessage(text)
}

// convertTools converts provider tools to OpenAI tool params.
func (p *Provider) convertTools(tools []providers.Tool) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, 0, len(tools))

	for _, tool := range tools {
		// Build parameters schema
		properties := make(map[string]interface{})
		for k, v := range tool.InputSchema.Properties {
			properties[k] = v
		}

		params := openai.FunctionParameters{
			"type":       "object",
			"properties": properties,
			"required":   tool.InputSchema.Required,
		}

		result = append(result, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  params,
			},
		})
	}

	return result
}

// convertResponse converts an OpenAI response to a provider response.
func (p *Provider) convertResponse(resp *openai.ChatCompletion) *providers.MessageResponse {
	if resp == nil || len(resp.Choices) == 0 {
		return &providers.MessageResponse{}
	}

	choice := resp.Choices[0]
	result := &providers.MessageResponse{
		StopReason: convertFinishReason(string(choice.FinishReason)),
	}

	// Add text content if present
	if choice.Message.Content != "" {
		result.Content = append(result.Content, providers.ContentBlock{
			Type: "text",
			Text: choice.Message.Content,
		})
	}

	// Add tool calls if present
	for _, tc := range choice.Message.ToolCalls {
		result.Content = append(result.Content, providers.ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	return result
}

// convertFinishReason converts OpenAI finish reason to provider stop reason.
func convertFinishReason(reason string) providers.StopReason {
	switch reason {
	case "stop":
		return providers.StopReasonEndTurn
	case "tool_calls":
		return providers.StopReasonToolUse
	case "length":
		return providers.StopReasonMaxTokens
	case "content_filter":
		return providers.StopReasonStopSequence
	default:
		return providers.StopReason(reason)
	}
}

// Package gemini provides a Google Gemini API implementation of the Provider interface.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/lex00/wetwire-core-go/providers"
	"google.golang.org/api/option"
)

// DefaultModel is the default model used by the Gemini provider.
const DefaultModel = "gemini-1.5-flash"

// Provider implements the providers.Provider interface using the Google Gemini API.
type Provider struct {
	client *genai.Client
}

// Config contains configuration for the Gemini provider.
type Config struct {
	// APIKey for Google AI Studio (defaults to GEMINI_API_KEY env var)
	APIKey string
}

// New creates a new Gemini provider.
func New(config Config) (*Provider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Provider{
		client: client,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "gemini"
}

// CreateMessage sends a message request and returns the complete response.
func (p *Provider) CreateMessage(ctx context.Context, req providers.MessageRequest) (*providers.MessageResponse, error) {
	model := p.getModel(req.Model)

	// Configure generation
	p.configureGeneration(model, req)

	// Build chat session
	session := p.buildSession(model, req)

	// Send the last user message
	lastMessage := req.Messages[len(req.Messages)-1]
	if lastMessage.Role != "user" {
		return nil, fmt.Errorf("last message must be from user")
	}

	// Extract text from content blocks
	var prompt string
	for _, block := range lastMessage.Content {
		if block.Type == "text" {
			prompt = block.Text
			break
		}
	}

	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	return p.convertResponse(resp), nil
}

// StreamMessage sends a message request and streams the response via the handler.
func (p *Provider) StreamMessage(ctx context.Context, req providers.MessageRequest, handler providers.StreamHandler) (*providers.MessageResponse, error) {
	model := p.getModel(req.Model)

	// Configure generation
	p.configureGeneration(model, req)

	// Build chat session
	session := p.buildSession(model, req)

	// Send the last user message
	lastMessage := req.Messages[len(req.Messages)-1]
	if lastMessage.Role != "user" {
		return nil, fmt.Errorf("last message must be from user")
	}

	// Extract text from content blocks
	var prompt string
	for _, block := range lastMessage.Content {
		if block.Type == "text" {
			prompt = block.Text
			break
		}
	}

	iter := session.SendMessageStream(ctx, genai.Text(prompt))

	var fullText strings.Builder
	var functionCalls []*genai.FunctionCall

	for {
		resp, err := iter.Next()
		if err != nil {
			if err.Error() == "iterator done" || strings.Contains(err.Error(), "EOF") {
				break
			}
			return nil, fmt.Errorf("streaming failed: %w", err)
		}

		for _, candidate := range resp.Candidates {
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					if text, ok := part.(genai.Text); ok {
						textStr := string(text)
						handler(textStr)
						fullText.WriteString(textStr)
					}
					if fc, ok := part.(genai.FunctionCall); ok {
						functionCalls = append(functionCalls, &fc)
					}
				}
			}
		}
	}

	// Build response
	response := &providers.MessageResponse{
		StopReason: providers.StopReasonEndTurn,
	}

	// Add text content if present
	if fullText.Len() > 0 {
		response.Content = append(response.Content, providers.ContentBlock{
			Type: "text",
			Text: fullText.String(),
		})
	}

	// Add function calls if present
	for i, fc := range functionCalls {
		argsJSON, _ := json.Marshal(fc.Args)
		response.Content = append(response.Content, providers.ContentBlock{
			Type:  "tool_use",
			ID:    fmt.Sprintf("call_%d", i),
			Name:  fc.Name,
			Input: argsJSON,
		})
		response.StopReason = providers.StopReasonToolUse
	}

	return response, nil
}

// getModel returns a configured generative model.
func (p *Provider) getModel(modelName string) *genai.GenerativeModel {
	if modelName == "" {
		modelName = DefaultModel
	}
	return p.client.GenerativeModel(modelName)
}

// configureGeneration configures the model's generation parameters.
func (p *Provider) configureGeneration(model *genai.GenerativeModel, req providers.MessageRequest) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	model.SetMaxOutputTokens(int32(maxTokens))

	// Set system instruction if provided
	if req.System != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(req.System)},
		}
	}

	// Configure tools if present
	if len(req.Tools) > 0 {
		model.Tools = []*genai.Tool{p.convertTools(req.Tools)}
	}
}

// buildSession builds a chat session with history.
func (p *Provider) buildSession(model *genai.GenerativeModel, req providers.MessageRequest) *genai.ChatSession {
	session := model.StartChat()

	// Add conversation history (excluding the last message which we'll send separately)
	if len(req.Messages) > 1 {
		for _, msg := range req.Messages[:len(req.Messages)-1] {
			content := p.convertMessageToContent(msg)
			session.History = append(session.History, content)
		}
	}

	return session
}

// convertMessageToContent converts a provider message to Gemini content.
func (p *Provider) convertMessageToContent(msg providers.Message) *genai.Content {
	var parts []genai.Part

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			parts = append(parts, genai.Text(block.Text))
		case "tool_use":
			// Convert to function call
			var args map[string]any
			if len(block.Input) > 0 {
				_ = json.Unmarshal(block.Input, &args)
			}
			parts = append(parts, genai.FunctionCall{
				Name: block.Name,
				Args: args,
			})
		case "tool_result":
			// Convert to function response
			var result any = block.Content
			if !block.IsError {
				// Try to parse as JSON
				var parsed any
				if err := json.Unmarshal([]byte(block.Content), &parsed); err == nil {
					result = parsed
				}
			}
			parts = append(parts, genai.FunctionResponse{
				Name:     block.ToolUseID,
				Response: map[string]any{"result": result},
			})
		}
	}

	role := "user"
	if msg.Role == "assistant" {
		role = "model"
	}

	return &genai.Content{
		Role:  role,
		Parts: parts,
	}
}

// convertTools converts provider tools to Gemini tool.
func (p *Provider) convertTools(tools []providers.Tool) *genai.Tool {
	functionDeclarations := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		// Convert properties to Gemini schema
		properties := make(map[string]*genai.Schema)
		for name, prop := range tool.InputSchema.Properties {
			// Create a schema for each property
			schema := &genai.Schema{
				Type: genai.TypeObject,
			}

			// Try to extract type information from the property
			if propMap, ok := prop.(map[string]any); ok {
				if typeStr, ok := propMap["type"].(string); ok {
					schema.Type = convertJSONTypeToGemini(typeStr)
				}
				if desc, ok := propMap["description"].(string); ok {
					schema.Description = desc
				}
			}

			properties[name] = schema
		}

		functionDeclarations = append(functionDeclarations, &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters: &genai.Schema{
				Type:       genai.TypeObject,
				Properties: properties,
				Required:   tool.InputSchema.Required,
			},
		})
	}

	return &genai.Tool{
		FunctionDeclarations: functionDeclarations,
	}
}

// convertJSONTypeToGemini converts JSON schema type to Gemini type.
func convertJSONTypeToGemini(jsonType string) genai.Type {
	switch jsonType {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeUnspecified
	}
}

// convertResponse converts a Gemini response to a provider response.
func (p *Provider) convertResponse(resp *genai.GenerateContentResponse) *providers.MessageResponse {
	if resp == nil || len(resp.Candidates) == 0 {
		return &providers.MessageResponse{}
	}

	candidate := resp.Candidates[0]
	result := &providers.MessageResponse{
		StopReason: providers.StopReasonEndTurn,
	}

	if candidate.Content != nil {
		for i, part := range candidate.Content.Parts {
			switch v := part.(type) {
			case genai.Text:
				result.Content = append(result.Content, providers.ContentBlock{
					Type: "text",
					Text: string(v),
				})
			case genai.FunctionCall:
				argsJSON, _ := json.Marshal(v.Args)
				result.Content = append(result.Content, providers.ContentBlock{
					Type:  "tool_use",
					ID:    fmt.Sprintf("call_%d", i),
					Name:  v.Name,
					Input: argsJSON,
				})
				result.StopReason = providers.StopReasonToolUse
			}
		}
	}

	return result
}

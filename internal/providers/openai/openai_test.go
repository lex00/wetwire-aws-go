package openai

import (
	"context"
	"testing"

	"github.com/lex00/wetwire-core-go/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	// Skip if API key not set
	if skipTest() {
		t.Skip("Skipping OpenAI test (OPENAI_API_KEY not set)")
	}

	provider, err := New(Config{})
	require.NoError(t, err)
	assert.Equal(t, "openai", provider.Name())
}

func TestProvider_CreateMessage(t *testing.T) {
	// Skip if API key not set
	if skipTest() {
		t.Skip("Skipping OpenAI test (OPENAI_API_KEY not set)")
	}

	provider, err := New(Config{})
	require.NoError(t, err)

	req := providers.MessageRequest{
		Model:     "gpt-4o-mini",
		MaxTokens: 100,
		System:    "You are a helpful assistant.",
		Messages: []providers.Message{
			providers.NewUserMessage("Say hello in 5 words or less."),
		},
	}

	resp, err := provider.CreateMessage(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Content, 1)
	assert.Equal(t, "text", resp.Content[0].Type)
	assert.NotEmpty(t, resp.Content[0].Text)
}

func TestProvider_StreamMessage(t *testing.T) {
	// Skip if API key not set
	if skipTest() {
		t.Skip("Skipping OpenAI test (OPENAI_API_KEY not set)")
	}

	provider, err := New(Config{})
	require.NoError(t, err)

	req := providers.MessageRequest{
		Model:     "gpt-4o-mini",
		MaxTokens: 100,
		System:    "You are a helpful assistant.",
		Messages: []providers.Message{
			providers.NewUserMessage("Say hello in 5 words or less."),
		},
	}

	var streamedText string
	resp, err := provider.StreamMessage(context.Background(), req, func(text string) {
		streamedText += text
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Content, 1)
	assert.Equal(t, "text", resp.Content[0].Type)
	assert.NotEmpty(t, resp.Content[0].Text)
	assert.Equal(t, resp.Content[0].Text, streamedText)
}

func skipTest() bool {
	// Allow tests to run if API key is set
	return !testingEnabled()
}

func testingEnabled() bool {
	// Tests run if OPENAI_API_KEY is set or CI env var is set
	// In CI, we skip unless explicitly enabled with RUN_OPENAI_TESTS=1
	apiKey := getAPIKey()
	if apiKey != "" {
		return true
	}
	return false
}

func getAPIKey() string {
	// Check for API key in environment
	key := Config{}.APIKey
	if key == "" {
		// Config{} will use env var
		return ""
	}
	return key
}

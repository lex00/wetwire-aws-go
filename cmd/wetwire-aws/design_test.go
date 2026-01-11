package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesignCommand_ProviderFlag(t *testing.T) {
	cmd := newDesignCmd()

	// Verify --provider flag exists
	providerFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, providerFlag, "provider flag should exist")
	assert.Equal(t, "anthropic", providerFlag.DefValue, "default provider should be anthropic")
	assert.Equal(t, "AI provider: 'anthropic' or 'kiro'", providerFlag.Usage)
}

func TestDesignCommand_KiroAgentName(t *testing.T) {
	// The agent name should be wetwire-aws-runner (domain-specific)
	// This is configured in internal/kiro/configs/wetwire-aws-runner.json
	cmd := newDesignCmd()
	assert.Contains(t, cmd.Long, "wetwire-aws-runner", "help text should reference the correct agent name")
}

func TestDesignCommand_RequiredPromptForAnthropic(t *testing.T) {
	cmd := newDesignCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required for the anthropic provider")
}

func TestDesignCommand_OptionalPromptForKiro(t *testing.T) {
	// Kiro provider should allow empty prompt
	cmd := newDesignCmd()
	cmd.SetArgs([]string{"--provider", "kiro"})

	// This will fail with kiro-cli not found, but that's expected
	// The point is it shouldn't fail with "prompt required" error
	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "prompt is required")
	}
}

func TestDesignCommand_UnknownProvider(t *testing.T) {
	cmd := newDesignCmd()
	cmd.SetArgs([]string{"--provider", "unknown", "test prompt"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

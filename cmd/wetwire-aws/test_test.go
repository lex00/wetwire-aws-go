package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestCommand_ProviderFlag(t *testing.T) {
	cmd := newTestCmd()

	// Verify --provider flag exists
	providerFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, providerFlag, "provider flag should exist")
	assert.Equal(t, "anthropic", providerFlag.DefValue, "default provider should be anthropic")
	assert.Equal(t, "AI provider: 'anthropic' or 'kiro'", providerFlag.Usage)
}

func TestTestCommand_PersonaFlag(t *testing.T) {
	cmd := newTestCmd()

	personaFlag := cmd.Flags().Lookup("persona")
	require.NotNil(t, personaFlag, "persona flag should exist")
	assert.Equal(t, "intermediate", personaFlag.DefValue, "default persona should be intermediate")
}

func TestTestCommand_AllPersonasFlag(t *testing.T) {
	cmd := newTestCmd()

	allPersonasFlag := cmd.Flags().Lookup("all-personas")
	require.NotNil(t, allPersonasFlag, "all-personas flag should exist")
	assert.Equal(t, "false", allPersonasFlag.DefValue)
}

func TestTestCommand_ScenarioFlag(t *testing.T) {
	cmd := newTestCmd()

	scenarioFlag := cmd.Flags().Lookup("scenario")
	require.NotNil(t, scenarioFlag, "scenario flag should exist")
	assert.Equal(t, "default", scenarioFlag.DefValue)
}

func TestTestCommand_RequiresPrompt(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.Error(t, err)
	// Cobra requires at least 1 arg
	assert.Contains(t, err.Error(), "requires at least 1 arg")
}

func TestTestCommand_UnknownProvider(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--provider", "unknown", "test prompt"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestTestCommand_SkipsKiroWhenEnvSet(t *testing.T) {
	t.Setenv("SKIP_KIRO_TESTS", "1")

	cmd := newTestCmd()
	cmd.SetArgs([]string{"--provider", "kiro", "test prompt"})

	// Should not error when SKIP_KIRO_TESTS is set
	err := cmd.Execute()
	assert.NoError(t, err)
}

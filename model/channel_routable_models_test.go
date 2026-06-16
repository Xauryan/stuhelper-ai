package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelGetRoutableModelsIncludesModelMappingSources(t *testing.T) {
	modelMapping := `{
		" claude-sonnet-4-6 ": "anthropic.claude-sonnet-4-6",
		"gpt-4o": "upstream-gpt-4o",
		"empty-target": " "
	}`
	channel := Channel{
		Models:       "upstream-gpt-4o, gpt-4o,,anthropic.claude-sonnet-4-6",
		ModelMapping: &modelMapping,
	}

	require.Equal(t, []string{
		"upstream-gpt-4o",
		"gpt-4o",
		"anthropic.claude-sonnet-4-6",
		"claude-sonnet-4-6",
	}, channel.GetRoutableModels())
}

func TestChannelGetRoutableModelsIgnoresInvalidModelMapping(t *testing.T) {
	modelMapping := `{invalid`
	channel := Channel{
		Models:       "gpt-4o-mini",
		ModelMapping: &modelMapping,
	}

	require.Equal(t, []string{"gpt-4o-mini"}, channel.GetRoutableModels())
}

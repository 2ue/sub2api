package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountSupportsOpenAIImageCapability_APIKeyOnly(t *testing.T) {
	apiKeyAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	oauthAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	require.True(t, apiKeyAccount.SupportsOpenAIImageCapability(OpenAIImagesCapabilityBasic))
	require.True(t, apiKeyAccount.SupportsOpenAIImageCapability(OpenAIImagesCapabilityNative))
	require.False(t, oauthAccount.SupportsOpenAIImageCapability(OpenAIImagesCapabilityBasic))
	require.False(t, oauthAccount.SupportsOpenAIImageCapability(OpenAIImagesCapabilityNative))
}

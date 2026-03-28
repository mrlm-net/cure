package gemini

import (
	"net/http"

	"github.com/mrlm-net/cure/pkg/agent"
)

// NewAdapterForTest creates a geminiAdapter with the given configuration for testing.
// This bypasses the environment variable lookup used by NewGeminiAgent.
// It is only compiled into test binaries.
func NewAdapterForTest(apiKey, model string, maxTokens int, baseURL string, client *http.Client) agent.Agent {
	if client == nil {
		client = &http.Client{}
	}
	return &geminiAdapter{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		baseURL:   baseURL,
		client:    client,
	}
}

// SanitiseError exposes sanitiseError for white-box testing.
// It is only compiled into test binaries.
func SanitiseError(apiKey string, err error) string {
	a := &geminiAdapter{apiKey: apiKey}
	return a.sanitiseError(err)
}

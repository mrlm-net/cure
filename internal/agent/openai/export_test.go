package openai

// SanitiseError exposes sanitiseError for use in package tests.
var SanitiseError = sanitiseError

// NewAdapterForTest creates an openaiAdapter bypassing environment variable lookup.
// baseURL should point to a httptest.Server for isolated testing.
func NewAdapterForTest(apiKey, baseURL, model string) *openaiAdapter {
	return newTestAdapter(apiKey, baseURL, model)
}

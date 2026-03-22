package agent

// EstimateTokens returns a rough estimate of the token count for the given text.
// It uses the heuristic of 1 token ≈ 4 characters, which is accurate enough for
// context window budget calculations. For precise counts, use [Agent.CountTokens].
func EstimateTokens(text string) int {
	return len(text) / 4
}

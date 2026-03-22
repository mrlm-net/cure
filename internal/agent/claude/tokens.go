package claude

import (
	"context"
	"errors"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/mrlm-net/cure/pkg/agent"
)

// CountTokens returns the token count for the session using the Anthropic API.
// Returns agent.ErrCountNotSupported (wrapped) if the endpoint returns HTTP 404.
// Other errors (401, 529, network) are propagated as-is.
func (a *claudeAdapter) CountTokens(ctx context.Context, sess *agent.Session) (int, error) {
	msgParams := a.buildParams(sess)

	countParams := anthropic.MessageCountTokensParams{
		Model:    msgParams.Model,
		Messages: msgParams.Messages,
	}
	if len(msgParams.System) > 0 {
		countParams.System = anthropic.MessageCountTokensParamsSystemUnion{
			OfTextBlockArray: msgParams.System,
		}
	}

	resp, err := a.client.Messages.CountTokens(ctx, countParams)
	if err != nil {
		return 0, mapCountError(err)
	}
	return int(resp.InputTokens), nil
}

// mapCountError maps Anthropic API errors to agent sentinel errors.
func mapCountError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return fmt.Errorf("claude: count tokens endpoint unavailable: %w", agent.ErrCountNotSupported)
	}
	return err
}

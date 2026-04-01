package ctxcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// ResumeCommand implements "cure context resume <session-id>".
// It loads an existing session by ID and either sends a single turn (with
// --message or piped stdin) or enters interactive REPL mode.
type ResumeCommand struct {
	store agent.SessionStore

	// Flags
	message   string
	format    string
	model     string
	maxTokens int
	skillName string
}

func (c *ResumeCommand) Name() string        { return "resume" }
func (c *ResumeCommand) Description() string { return "Resume an existing AI conversation session" }

func (c *ResumeCommand) Usage() string {
	return `Usage: cure context resume <session-id> [options]

Resume a previously saved AI conversation session. The session history is
preserved across invocations. With --message the command sends a single turn
and exits. Without --message and connected to a terminal the command enters
an interactive REPL. Piped stdin is read as a single message.

Arguments:
  <session-id>    ID of the session to resume (required)

Flags:
  --message       User message to send (optional; triggers single-turn mode)
  --format        Output format: "text" (default) or "ndjson"
  --skill         Named skill to activate (or switch) for this session
  --model         Model name override for this turn (provider-specific; uses provider default if empty)
  --max-tokens    Maximum tokens override for this turn (uses provider default if 0)

Examples:
  cure context resume abc123def456
  cure context resume abc123def456 --message "Continue where we left off"
  cure context resume abc123def456 --skill go-expert --message "Help me optimize this"
  cure context resume abc123def456 --model "gpt-4o-mini" --message "Quick follow-up"
  echo "What was the last thing we discussed?" | cure context resume abc123def456
`
}

func (c *ResumeCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("context-resume", flag.ContinueOnError)
	fs.StringVar(&c.message, "message", "", "User message to send")
	fs.StringVar(&c.format, "format", "text", `Output format: "text" or "ndjson"`)
	fs.StringVar(&c.model, "model", "", "Model name override for this turn (provider-specific; uses provider default if empty)")
	fs.IntVar(&c.maxTokens, "max-tokens", 0, "Maximum tokens override for this turn (uses provider default if 0)")
	fs.StringVar(&c.skillName, "skill", "", "Named skill to activate (or switch) for this session")
	return fs
}

func (c *ResumeCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("context resume: missing <session-id> argument")
	}
	sessionID := tc.Args[0]

	sess, err := c.store.Load(ctx, sessionID)
	if err != nil {
		if errors.Is(err, agent.ErrSessionNotFound) {
			return fmt.Errorf("context resume: session %q not found", sessionID)
		}
		return fmt.Errorf("context resume: %w", err)
	}

	cfg := map[string]any{}
	if c.model != "" {
		cfg["model"] = c.model
	}
	if c.maxTokens > 0 {
		cfg["max_tokens"] = c.maxTokens
	}

	a, err := agent.New(sess.Provider, cfg)
	if err != nil {
		return fmt.Errorf("context resume: %w", err)
	}

	// Do NOT update sess.Model on resume — preserve the stored model.
	// The agent's internal model setting changes but the session record does not.

	// --skill on resume overwrites the system prompt, tools, and skill name,
	// allowing callers to activate or switch skills on existing sessions.
	// Note: History is preserved — prior turns may be inconsistent with the
	// new skill's prompt if the skill is switched mid-session.
	if c.skillName != "" {
		skill, ok := agent.LookupSkill(c.skillName)
		if !ok {
			return fmt.Errorf("context resume: skill %q not registered", c.skillName)
		}
		sess.SystemPrompt = skill.SystemPrompt
		sess.Tools = skill.Tools
		sess.SkillName = skill.Name
	}

	return runTurn(ctx, tc, a, c.store, sess, c.message, c.format)
}

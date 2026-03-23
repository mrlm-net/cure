package ctxcmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// runREPL runs an interactive read-evaluate-print loop that reads user input
// line by line from reader, sends each line to the agent, and streams the
// response back to the user.
//
// Special commands:
//   - "/exit" or "/quit" — terminate the REPL cleanly.
//   - "/fork" — fork the current session, print the new session ID, and
//     continue the REPL with the forked session.
//
// Empty lines are skipped. EOF with no pending input exits cleanly.
// On a provider error the user message is rolled back from the history so the
// session remains consistent.
func runREPL(
	ctx context.Context,
	tc *terminal.Context,
	a agent.Agent,
	st agent.SessionStore,
	sess *agent.Session,
	format string,
	reader io.Reader,
) error {
	br := bufio.NewReader(reader)

	for {
		// Print the prompt on stderr so it does not pollute the text output
		// stream (which may be captured or redirected by the caller).
		fmt.Fprint(tc.Stderr, "> ")

		line, err := br.ReadString('\n')

		// Handle EOF: if there is content on the last line, process it then stop.
		if err == io.EOF {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if replErr := handleREPLLine(ctx, tc, a, st, sess, format, trimmed); replErr != nil {
					fmt.Fprintf(tc.Stderr, "error: %v\n", replErr)
				}
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("context: repl read: %w", err)
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for built-in REPL commands.
		switch strings.ToLower(trimmed) {
		case "/exit", "/quit":
			return nil

		case "/fork":
			forked, forkErr := st.Fork(ctx, sess.ID)
			if forkErr != nil {
				fmt.Fprintf(tc.Stderr, "error: fork failed: %v\n", forkErr)
				continue
			}
			sess = forked
			fmt.Fprintln(tc.Stdout, forked.ID)
			continue
		}

		// Regular user message.
		if replErr := handleREPLLine(ctx, tc, a, st, sess, format, trimmed); replErr != nil {
			fmt.Fprintf(tc.Stderr, "error: %v\n", replErr)
		}
	}
}

// handleREPLLine sends a single user message to the agent and streams the reply.
// On error the user message is rolled back from the history.
func handleREPLLine(
	ctx context.Context,
	tc *terminal.Context,
	a agent.Agent,
	st agent.SessionStore,
	sess *agent.Session,
	format string,
	msg string,
) error {
	sess.AppendUserMessage(msg)
	events := a.Run(ctx, sess)
	responseText, err := dispatch(ctx, tc, format, events)
	if err != nil {
		sess.History = sess.History[:len(sess.History)-1]
		return err
	}
	sess.AppendAssistantMessage(responseText)
	// Non-fatal save failure: log and continue.
	if saveErr := st.Save(ctx, sess); saveErr != nil {
		fmt.Fprintf(tc.Stderr, "warning: failed to save session: %v\n", saveErr)
	}
	// Blank line after each assistant reply for visual separation.
	fmt.Fprintln(tc.Stdout)
	return nil
}

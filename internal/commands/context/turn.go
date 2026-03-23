package ctxcmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// stdinTTY reports whether the effective stdin is an interactive terminal.
// If tc.Stdin is injected (e.g. by PipelineRunner), it is always treated as
// non-TTY piped input regardless of os.Stdin's file-descriptor state.
func stdinTTY(tc *terminal.Context) bool {
	if tc.Stdin != nil {
		return false
	}
	return isatty(os.Stdin.Fd())
}

// runTurn executes a single conversation turn or starts a REPL, depending on
// how input is available:
//
//  1. msg is non-empty — single turn with that message.
//  2. stdin is not a TTY (piped) — read all of stdin as the message, single turn.
//  3. stdin is a TTY and format is "ndjson" — return a usage error (REPL is not
//     supported in NDJSON mode).
//  4. stdin is a TTY and msg is empty — enter interactive REPL mode.
func runTurn(
	ctx context.Context,
	tc *terminal.Context,
	a agent.Agent,
	st agent.SessionStore,
	sess *agent.Session,
	msg string,
	format string,
) error {
	return doRunTurn(ctx, tc, a, st, sess, msg, format, stdinTTY(tc))
}

// doRunTurn is the testable core of runTurn. The tty parameter controls whether
// stdin is treated as an interactive terminal, allowing tests to exercise all
// branches without a real PTY.
func doRunTurn(
	ctx context.Context,
	tc *terminal.Context,
	a agent.Agent,
	st agent.SessionStore,
	sess *agent.Session,
	msg string,
	format string,
	tty bool,
) error {
	stdin := stdinReader(tc)

	// Branch 1: explicit message provided via flag.
	if msg != "" {
		return executeSingleTurn(ctx, tc, a, st, sess, msg, format)
	}

	// Branch 2: stdin is not a TTY — read piped input as the message.
	if !tty {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("context: read stdin: %w", err)
		}
		msg = string(data)
		if msg == "" {
			return fmt.Errorf("context: no message provided and stdin is empty")
		}
		return executeSingleTurn(ctx, tc, a, st, sess, msg, format)
	}

	// Branch 3: TTY + NDJSON + no message → unsupported combination.
	if format == "ndjson" {
		return fmt.Errorf("context: --format ndjson requires --message or piped stdin; REPL mode is not supported with NDJSON output")
	}

	// Branch 4: TTY + text format + no message → REPL.
	return runREPL(ctx, tc, a, st, sess, format, stdin)
}

// executeSingleTurn appends msg as a user message, runs the agent, streams the
// response, appends the assistant reply, and saves the session.
func executeSingleTurn(
	ctx context.Context,
	tc *terminal.Context,
	a agent.Agent,
	st agent.SessionStore,
	sess *agent.Session,
	msg string,
	format string,
) error {
	sess.AppendUserMessage(msg)
	events := a.Run(ctx, sess)
	responseText, err := dispatch(ctx, tc, format, events)
	if err != nil {
		// Roll back the user message we just appended so the session remains clean.
		sess.History = sess.History[:len(sess.History)-1]
		return err
	}
	sess.AppendAssistantMessage(responseText)
	if saveErr := st.Save(ctx, sess); saveErr != nil {
		// Non-fatal: warn but do not abort — the response was already displayed.
		fmt.Fprintf(tc.Stderr, "warning: failed to save session: %v\n", saveErr)
	}
	return nil
}

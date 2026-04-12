// Package backlogcmd implements the "cure backlog" command group.
package backlogcmd

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/mrlm-net/cure/internal/backlog"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewBacklogCommand returns the "backlog" parent command.
func NewBacklogCommand(tracker backlog.Tracker) terminal.Command {
	router := terminal.New(
		terminal.WithName("backlog"),
		terminal.WithDescription("Manage work items (issues, tickets)"),
	)
	router.Register(&listCmd{tracker: tracker})
	router.Register(&viewCmd{tracker: tracker})
	router.Register(&createCmd{tracker: tracker})
	router.Register(&closeCmd{tracker: tracker})
	return router
}

// --- list ---
type listCmd struct {
	tracker backlog.Tracker
	state   string
	limit   int
}

func (c *listCmd) Name() string        { return "list" }
func (c *listCmd) Description() string { return "List work items" }
func (c *listCmd) Usage() string       { return "Usage: cure backlog list [--state open|closed|all] [--limit N]" }
func (c *listCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.StringVar(&c.state, "state", "open", "filter by state")
	fs.IntVar(&c.limit, "limit", 30, "max items")
	return fs
}

func (c *listCmd) Run(ctx context.Context, tc *terminal.Context) error {
	items, err := c.tracker.List(ctx, backlog.Filter{State: c.state, Limit: c.limit})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintln(tc.Stdout, "No items found.")
		return nil
	}
	tw := tabwriter.NewWriter(tc.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tSTATE\tLABELS")
	for _, i := range items {
		fmt.Fprintf(tw, "#%s\t%s\t%s\t%s\n", i.ID, truncate(i.Title, 60), i.State, strings.Join(i.Labels, ","))
	}
	return tw.Flush()
}

// --- view ---
type viewCmd struct{ tracker backlog.Tracker }

func (c *viewCmd) Name() string        { return "view" }
func (c *viewCmd) Description() string { return "View a work item" }
func (c *viewCmd) Usage() string       { return "Usage: cure backlog view <id>" }
func (c *viewCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("view", flag.ContinueOnError) }

func (c *viewCmd) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) < 1 {
		return fmt.Errorf("usage: cure backlog view <id>")
	}
	item, err := c.tracker.Get(ctx, tc.Args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(tc.Stdout, "#%s: %s [%s]\n\n%s\n", item.ID, item.Title, item.State, item.Body)
	return nil
}

// --- create ---
type createCmd struct {
	tracker backlog.Tracker
	title   string
	body    string
	labels  string
}

func (c *createCmd) Name() string        { return "create" }
func (c *createCmd) Description() string { return "Create a work item" }
func (c *createCmd) Usage() string {
	return "Usage: cure backlog create --title \"...\" [--body \"...\"] [--label \"...\"]"
}
func (c *createCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.StringVar(&c.title, "title", "", "item title")
	fs.StringVar(&c.body, "body", "", "item body")
	fs.StringVar(&c.labels, "label", "", "comma-separated labels")
	return fs
}

func (c *createCmd) Run(ctx context.Context, tc *terminal.Context) error {
	if c.title == "" {
		return fmt.Errorf("--title is required")
	}
	var labels []string
	if c.labels != "" {
		labels = strings.Split(c.labels, ",")
	}
	item, err := c.tracker.Create(ctx, &backlog.WorkItem{Title: c.title, Body: c.body, Labels: labels})
	if err != nil {
		return err
	}
	fmt.Fprintf(tc.Stdout, "Created: %s\n", item.URL)
	return nil
}

// --- close ---
type closeCmd struct{ tracker backlog.Tracker }

func (c *closeCmd) Name() string        { return "close" }
func (c *closeCmd) Description() string { return "Close a work item" }
func (c *closeCmd) Usage() string       { return "Usage: cure backlog close <id> [--comment \"...\"]" }
func (c *closeCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("close", flag.ContinueOnError) }

func (c *closeCmd) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) < 1 {
		return fmt.Errorf("usage: cure backlog close <id>")
	}
	return c.tracker.Close(ctx, tc.Args[0], "")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

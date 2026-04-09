// Package backlog provides a provider-agnostic interface for work item
// tracking (GitHub Issues, Azure DevOps Work Items).
package backlog

import "context"

// WorkItem is the provider-agnostic work item model.
type WorkItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Body        string   `json:"body,omitempty"`
	State       string   `json:"state"`
	Labels      []string `json:"labels,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	TrackerType string   `json:"tracker_type"`
	URL         string   `json:"url,omitempty"`
}

// WorkItemUpdate holds fields to update on a work item.
type WorkItemUpdate struct {
	State   string   `json:"state,omitempty"`
	Title   string   `json:"title,omitempty"`
	Body    string   `json:"body,omitempty"`
	Labels  []string `json:"labels,omitempty"`
	Comment string   `json:"comment,omitempty"`
}

// Filter constrains work item queries.
type Filter struct {
	State  string // "open", "closed", "all"
	Labels string // comma-separated
	Limit  int
}

// Tracker is the abstraction over work item backends.
type Tracker interface {
	List(ctx context.Context, filter Filter) ([]WorkItem, error)
	Get(ctx context.Context, id string) (*WorkItem, error)
	Create(ctx context.Context, item *WorkItem) (*WorkItem, error)
	Update(ctx context.Context, id string, changes *WorkItemUpdate) error
	Close(ctx context.Context, id string, comment string) error
}

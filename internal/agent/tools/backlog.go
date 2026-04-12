package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mrlm-net/cure/internal/backlog"
	"github.com/mrlm-net/cure/pkg/agent"
)

// BacklogTools returns agent.Tool implementations backed by a Tracker.
func BacklogTools(tracker backlog.Tracker) []agent.Tool {
	return []agent.Tool{
		agent.FuncTool("backlog_list", "List open work items from project tracker",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]any{"type": "string", "description": "open, closed, or all"},
					"limit": map[string]any{"type": "integer", "description": "max items"},
				},
			},
			func(ctx context.Context, args map[string]any) (string, error) {
				state, _ := args["state"].(string)
				if state == "" {
					state = "open"
				}
				limit := 20
				if v, ok := args["limit"].(float64); ok {
					limit = int(v)
				}
				items, err := tracker.List(ctx, backlog.Filter{State: state, Limit: limit})
				if err != nil {
					return "", err
				}
				data, _ := json.MarshalIndent(items, "", "  ")
				return string(data), nil
			},
		),

		agent.FuncTool("backlog_create", "Create a new work item in the project tracker",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":  map[string]any{"type": "string"},
					"body":   map[string]any{"type": "string"},
					"labels": map[string]any{"type": "string", "description": "comma-separated"},
				},
				"required": []string{"title"},
			},
			func(ctx context.Context, args map[string]any) (string, error) {
				title, _ := args["title"].(string)
				body, _ := args["body"].(string)
				labelsStr, _ := args["labels"].(string)
				var labels []string
				if labelsStr != "" {
					for _, l := range strings.Split(labelsStr, ",") {
						labels = append(labels, strings.TrimSpace(l))
					}
				}
				item, err := tracker.Create(ctx, &backlog.WorkItem{Title: title, Body: body, Labels: labels})
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Created #%s: %s", item.ID, item.URL), nil
			},
		),

		agent.FuncTool("backlog_view", "View a work item by ID",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			func(ctx context.Context, args map[string]any) (string, error) {
				id, _ := args["id"].(string)
				item, err := tracker.Get(ctx, id)
				if err != nil {
					return "", err
				}
				data, _ := json.MarshalIndent(item, "", "  ")
				return string(data), nil
			},
		),

		agent.FuncTool("backlog_update", "Update a work item state or add a comment",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "string"},
					"state":   map[string]any{"type": "string"},
					"comment": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			func(ctx context.Context, args map[string]any) (string, error) {
				id, _ := args["id"].(string)
				state, _ := args["state"].(string)
				comment, _ := args["comment"].(string)
				err := tracker.Update(ctx, id, &backlog.WorkItemUpdate{State: state, Comment: comment})
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Updated #%s", id), nil
			},
		),
	}
}

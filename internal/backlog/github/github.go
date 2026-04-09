// Package github implements the backlog.Tracker interface using the gh CLI.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mrlm-net/cure/internal/backlog"
)

// Tracker implements backlog.Tracker via gh CLI.
type Tracker struct {
	Owner string
	Repo  string
}

var _ backlog.Tracker = (*Tracker)(nil)

func (t *Tracker) repoArg() string {
	if t.Owner != "" && t.Repo != "" {
		return t.Owner + "/" + t.Repo
	}
	return ""
}

func (t *Tracker) List(_ context.Context, filter backlog.Filter) ([]backlog.WorkItem, error) {
	args := []string{"issue", "list", "--json", "number,title,state,labels,assignees,url"}
	if filter.State != "" && filter.State != "all" {
		args = append(args, "--state", filter.State)
	}
	if filter.Labels != "" {
		args = append(args, "--label", filter.Labels)
	}
	if filter.Limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", filter.Limit))
	}
	if repo := t.repoArg(); repo != "" {
		args = append(args, "--repo", repo)
	}

	out, err := gh(args...)
	if err != nil {
		return nil, err
	}

	var ghIssues []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		URL       string `json:"url"`
		Labels    []struct{ Name string } `json:"labels"`
		Assignees []struct{ Login string } `json:"assignees"`
	}
	if err := json.Unmarshal([]byte(out), &ghIssues); err != nil {
		return nil, fmt.Errorf("github: parse issues: %w", err)
	}

	items := make([]backlog.WorkItem, 0, len(ghIssues))
	for _, i := range ghIssues {
		var labels []string
		for _, l := range i.Labels {
			labels = append(labels, l.Name)
		}
		var assignee string
		if len(i.Assignees) > 0 {
			assignee = i.Assignees[0].Login
		}
		items = append(items, backlog.WorkItem{
			ID:          fmt.Sprintf("%d", i.Number),
			Title:       i.Title,
			State:       strings.ToLower(i.State),
			Labels:      labels,
			Assignee:    assignee,
			TrackerType: "github",
			URL:         i.URL,
		})
	}
	return items, nil
}

func (t *Tracker) Get(_ context.Context, id string) (*backlog.WorkItem, error) {
	args := []string{"issue", "view", id, "--json", "number,title,body,state,labels,assignees,url"}
	if repo := t.repoArg(); repo != "" {
		args = append(args, "--repo", repo)
	}

	out, err := gh(args...)
	if err != nil {
		return nil, err
	}

	var i struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		State     string `json:"state"`
		URL       string `json:"url"`
		Labels    []struct{ Name string } `json:"labels"`
		Assignees []struct{ Login string } `json:"assignees"`
	}
	if err := json.Unmarshal([]byte(out), &i); err != nil {
		return nil, err
	}

	var labels []string
	for _, l := range i.Labels {
		labels = append(labels, l.Name)
	}
	var assignee string
	if len(i.Assignees) > 0 {
		assignee = i.Assignees[0].Login
	}
	return &backlog.WorkItem{
		ID:          fmt.Sprintf("%d", i.Number),
		Title:       i.Title,
		Body:        i.Body,
		State:       strings.ToLower(i.State),
		Labels:      labels,
		Assignee:    assignee,
		TrackerType: "github",
		URL:         i.URL,
	}, nil
}

func (t *Tracker) Create(_ context.Context, item *backlog.WorkItem) (*backlog.WorkItem, error) {
	args := []string{"issue", "create", "--title", item.Title, "--body", item.Body}
	for _, l := range item.Labels {
		args = append(args, "--label", l)
	}
	if repo := t.repoArg(); repo != "" {
		args = append(args, "--repo", repo)
	}

	out, err := gh(args...)
	if err != nil {
		return nil, err
	}
	url := strings.TrimSpace(out)
	item.URL = url
	item.TrackerType = "github"
	return item, nil
}

func (t *Tracker) Update(_ context.Context, id string, changes *backlog.WorkItemUpdate) error {
	args := []string{"issue", "edit", id}
	if changes.Title != "" {
		args = append(args, "--title", changes.Title)
	}
	if changes.Body != "" {
		args = append(args, "--body", changes.Body)
	}
	for _, l := range changes.Labels {
		args = append(args, "--add-label", l)
	}
	if repo := t.repoArg(); repo != "" {
		args = append(args, "--repo", repo)
	}
	_, err := gh(args...)
	if err != nil {
		return err
	}

	if changes.Comment != "" {
		commentArgs := []string{"issue", "comment", id, "--body", changes.Comment}
		if repo := t.repoArg(); repo != "" {
			commentArgs = append(commentArgs, "--repo", repo)
		}
		_, err = gh(commentArgs...)
	}
	return err
}

func (t *Tracker) Close(_ context.Context, id string, comment string) error {
	if comment != "" {
		commentArgs := []string{"issue", "comment", id, "--body", comment}
		if repo := t.repoArg(); repo != "" {
			commentArgs = append(commentArgs, "--repo", repo)
		}
		gh(commentArgs...)
	}

	args := []string{"issue", "close", id}
	if repo := t.repoArg(); repo != "" {
		args = append(args, "--repo", repo)
	}
	_, err := gh(args...)
	return err
}

func gh(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh %s: %s", strings.Join(args[:2], " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

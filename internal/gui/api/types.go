// Package api provides the REST API layer for the cure GUI server.
//
// All endpoints are mounted under /api/ and return application/json responses.
// Errors use a standard [ErrorResponse] envelope. Handlers receive shared
// dependencies via the [Deps] struct, which is injected at router construction.
package api

import "time"

// ErrorResponse is the standard JSON error envelope returned by all API
// endpoints when an error occurs.
type ErrorResponse struct {
	Error string `json:"error"`
}

// CheckResultResponse represents a single doctor check outcome.
type CheckResultResponse struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass" | "warn" | "fail"
	Message string `json:"message"`
}

// SessionSummary represents a session in list responses.
type SessionSummary struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ForkOf    string    `json:"fork_of,omitempty"`
	Turns     int       `json:"turns"`
}

// SessionDetail includes full message history alongside the session summary.
type SessionDetail struct {
	SessionSummary
	History []MessageResponse `json:"history"`
}

// MessageResponse represents a single turn in a session's conversation history.
type MessageResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CreateSessionRequest is the POST body for creating new sessions.
type CreateSessionRequest struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

// MessageRequest is the POST body for sending a message within a session.
type MessageRequest struct {
	Message string `json:"message"`
}

// GenerateRequest is the POST body for template generation.
type GenerateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
}

// GenerateResponse contains the rendered template output.
type GenerateResponse struct {
	Template string `json:"template"`
	Content  string `json:"content"`
}

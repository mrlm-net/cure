// Package agent provides a provider-agnostic interface for AI agent context management.
//
// Providers (Claude, OpenAI, local LLMs) are registered by name via [Register]
// and instantiated on demand via [New]. Provider adapters live in internal/agent/<provider>/
// and self-register via init() using the blank-import driver pattern.
//
// Streaming responses use [iter.Seq2] from the standard library (Go 1.23+).
// Sessions are persisted through the [SessionStore] interface — concrete implementations
// (e.g. JSON file store) are provided by other packages in this module.
package agent

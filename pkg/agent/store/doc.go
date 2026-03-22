// Package store provides concrete [agent.SessionStore] implementations.
//
// The [JSONStore] type stores each session as a JSON file under a configurable
// directory. Writes are atomic: data is first written to a temp file in the same
// directory as the target, then renamed into place, guaranteeing readers always
// see a complete file.
//
// The package is safe for concurrent use.
package store

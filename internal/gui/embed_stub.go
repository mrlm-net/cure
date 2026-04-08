//go:build no_frontend

package gui

import "io/fs"

// distFS is an empty filesystem used when the frontend is not built.
// This stub allows the package to compile without the frontend build artifacts.
var distFS fs.FS = emptyFS{}

type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

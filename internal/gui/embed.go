//go:build !no_frontend

package gui

import "embed"

//go:embed dist
var distFS embed.FS

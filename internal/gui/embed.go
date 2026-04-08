//go:build !no_frontend

package gui

import "embed"

//go:embed all:dist
var distFS embed.FS

package api

import (
	"net/http"

	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
)

// Deps holds the dependencies shared by all API handlers.
// Fields are injected at construction time and must not be nil.
type Deps struct {
	// Config is the merged application configuration, serialized as-is
	// by the config endpoint. ConfigObject is used (rather than *config.Config)
	// because the Config type's internal data field is unexported and cannot
	// be marshaled to JSON directly.
	Config config.ConfigObject

	// Checks is the list of doctor checks to execute on the doctor endpoint.
	Checks []doctor.CheckFunc

	// Port is the TCP port the GUI server is listening on.
	Port int
}

// NewAPIRouter returns an http.Handler that mounts all /api/* routes.
// Go 1.22+ ServeMux method+path patterns enforce allowed HTTP methods —
// requests with disallowed methods receive 405 Method Not Allowed automatically.
func NewAPIRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", healthHandler(deps.Port))
	mux.HandleFunc("GET /api/config", configHandler(deps.Config))
	mux.HandleFunc("GET /api/doctor", doctorHandler(deps.Checks))

	return mux
}

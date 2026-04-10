package api

import (
	"net/http"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/project"
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

	// Store is the session persistence layer used by CRUD and SSE endpoints.
	// When nil, session endpoints return 501 Not Implemented.
	Store agent.SessionStore

	// AgentRun is an optional function that runs an agent turn on a session
	// and streams results. When nil, the messages endpoint uses a built-in
	// echo stub that reflects the user's message back as word-level tokens.
	AgentRun AgentRunFunc

	// ProjectName is the auto-detected project name for the current cwd.
	// Used to associate new sessions with the project.
	ProjectName string

	// ProjectStore is the project persistence layer. When nil, project
	// endpoints return 501 Not Implemented.
	ProjectStore project.ProjectStore

	// ProjectRoots are the allowed file API root directories (from project repos).
	ProjectRoots []string
}

// NewAPIRouter returns an http.Handler that mounts all /api/* routes.
// Go 1.22+ ServeMux method+path patterns enforce allowed HTTP methods —
// requests with disallowed methods receive 405 Method Not Allowed automatically.
func NewAPIRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", healthHandler(deps.Port))
	mux.HandleFunc("GET /api/config", configHandler(deps.Config))
	mux.HandleFunc("GET /api/doctor", doctorHandler(deps.Checks))
	mux.HandleFunc("GET /api/doctor/platform", doctorHandler(doctor.ControlPlaneChecks()))
	mux.HandleFunc("GET /api/generate/list", generateListHandler())
	mux.HandleFunc("POST /api/generate/{template}", generateRunHandler())

	// Session endpoints require a store. When absent, all session routes
	// return 501 to signal that the feature is unavailable.
	if deps.Store != nil {
		defaults := configDefaults{
			defaultProvider: configString(deps.Config, "agent.provider"),
			defaultModel:    configString(deps.Config, "agent.model"),
		}

		mux.HandleFunc("GET /api/context/sessions", sessionsListHandler(deps.Store))
		mux.HandleFunc("POST /api/context/sessions", sessionsCreateHandler(deps.Store, defaults, deps.ProjectName))
		mux.HandleFunc("GET /api/context/sessions/{id}", sessionsGetHandler(deps.Store))
		mux.HandleFunc("DELETE /api/context/sessions/{id}", sessionsDeleteHandler(deps.Store))
		mux.HandleFunc("POST /api/context/sessions/{id}/fork", sessionsForkHandler(deps.Store))
		mux.HandleFunc("POST /api/context/sessions/{id}/messages", messagesHandler(deps.Store, deps.AgentRun))
	}

	// Project endpoints
	if deps.ProjectStore != nil {
		mux.HandleFunc("GET /api/project", projectListHandler(deps.ProjectStore))
		mux.HandleFunc("GET /api/project/{name}", projectGetHandler(deps.ProjectStore))
		mux.HandleFunc("PUT /api/project/{name}", projectUpdateHandler(deps.ProjectStore))
	}

	// Config update API
	mux.HandleFunc("PUT /api/config", configUpdateHandler())

	// Global settings API (form-friendly)
	mux.HandleFunc("GET /api/settings", settingsGetHandler())
	mux.HandleFunc("PUT /api/settings", settingsPutHandler())

	// File API (scoped to project roots)
	mux.HandleFunc("GET /api/editor/roots", fileRootsHandler(deps.ProjectRoots))
	mux.HandleFunc("GET /api/files", filesListHandler(deps.ProjectRoots))
	mux.HandleFunc("PUT /api/files", fileWriteQueryHandler(deps.ProjectRoots))
	mux.HandleFunc("GET /api/files/{path...}", fileReadHandler(deps.ProjectRoots))
	mux.HandleFunc("PUT /api/files/{path...}", fileWriteHandler(deps.ProjectRoots))

	return mux
}

// configString extracts a dot-notation string value from ConfigObject.
// Returns empty string when the key is absent or not a string.
func configString(cfg config.ConfigObject, key string) string {
	if cfg == nil {
		return ""
	}
	v, ok := cfg[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

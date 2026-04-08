package api

import (
	"encoding/json"
	"net/http"

	"github.com/mrlm-net/cure/pkg/doctor"
)

// doctorHandler returns a handler that executes all registered doctor checks
// and returns their results as a JSON array. Each check is run synchronously
// in order; panicking checks are recovered by the doctor package's own
// safety wrapper when using [doctor.Run], but here we call CheckFunc directly
// and rely on the handler-level recovery if needed.
func doctorHandler(checks []doctor.CheckFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		results := make([]CheckResultResponse, 0, len(checks))
		for _, check := range checks {
			r := check()
			results = append(results, CheckResultResponse{
				Name:    r.Name,
				Status:  checkStatusString(r.Status),
				Message: r.Message,
			})
		}

		_ = json.NewEncoder(w).Encode(results)
	}
}

// checkStatusString converts a doctor.CheckStatus to its JSON-friendly
// string representation.
func checkStatusString(s doctor.CheckStatus) string {
	switch s {
	case doctor.CheckPass:
		return "pass"
	case doctor.CheckWarn:
		return "warn"
	case doctor.CheckFail:
		return "fail"
	default:
		return "unknown"
	}
}

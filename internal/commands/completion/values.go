package completion

// FlagValues maps flag names to their valid values for shell completion.
// This is a static map for v0.4.0. Future versions may make this dynamic
// via Command metadata or runtime introspection.
var FlagValues = map[string][]string{
	"format": {"json", "html"},
	"method": {"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
}

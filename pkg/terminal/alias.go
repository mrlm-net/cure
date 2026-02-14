package terminal

// AliasProvider is an optional interface that commands can implement to
// declare their preferred aliases. The Router checks for this interface
// during Register() and automatically registers the aliases.
//
// This is optional -- commands that do not implement AliasProvider still
// work via Register(). Use [Router.RegisterWithAliases] for commands that do not
// implement this interface.
type AliasProvider interface {
	// Aliases returns the alias names for this command.
	// Each alias is registered as a route to the same command.
	Aliases() []string
}

// AliasRegistry extends CommandRegistry with alias awareness.
// [Router] implements this interface.
type AliasRegistry interface {
	CommandRegistry
	AliasesFor(name string) []string
}

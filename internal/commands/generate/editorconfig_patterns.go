package generate

// EditorSection represents a single [glob] section in an .editorconfig file.
type EditorSection struct {
	Glob                   string
	IndentStyle            string // "space" or "tab"
	IndentSize             string // "2", "4", or "tab"
	EndOfLine              string // "lf" or "crlf"
	Charset                string // "utf-8"
	TrimTrailingWhitespace string // "true" or "false"
	InsertFinalNewline     string // "true"
}

// editorConfigRules maps language keys to their EditorSection configuration.
var editorConfigRules = map[string]EditorSection{
	"go": {
		Glob:                   "*.go",
		IndentStyle:            "tab",
		IndentSize:             "tab",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"javascript": {
		Glob:                   "*.{js,jsx,ts,tsx,mjs,cjs}",
		IndentStyle:            "space",
		IndentSize:             "2",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"python": {
		Glob:                   "*.py",
		IndentStyle:            "space",
		IndentSize:             "4",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"rust": {
		Glob:                   "*.rs",
		IndentStyle:            "space",
		IndentSize:             "4",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"java": {
		Glob:                   "*.java",
		IndentStyle:            "space",
		IndentSize:             "4",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"shell": {
		Glob:                   "*.{sh,bash,zsh}",
		IndentStyle:            "space",
		IndentSize:             "2",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"markdown": {
		Glob:                   "*.{md,mdx}",
		IndentStyle:            "space",
		IndentSize:             "2",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "false",
		InsertFinalNewline:     "true",
	},
	"yaml": {
		Glob:                   "*.{yml,yaml}",
		IndentStyle:            "space",
		IndentSize:             "2",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
	"generic": {
		Glob:                   "*",
		IndentStyle:            "space",
		IndentSize:             "2",
		EndOfLine:              "lf",
		Charset:                "utf-8",
		TrimTrailingWhitespace: "true",
		InsertFinalNewline:     "true",
	},
}

// editorConfigLanguageOrder defines the canonical display order for the MultiSelect menu
// and for section rendering in the generated file.
var editorConfigLanguageOrder = []string{
	"go", "javascript", "python", "rust", "java", "shell", "markdown", "yaml", "generic",
}

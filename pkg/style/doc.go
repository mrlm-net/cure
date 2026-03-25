// Package style provides minimal ANSI terminal styling with NO_COLOR support.
//
// All functions are standalone — there is no struct or object to initialize.
// Styling is enabled by default and respects the NO_COLOR environment variable
// (see https://no-color.org): if NO_COLOR is set to any non-empty value at
// program startup, all styling functions return their input unchanged.
//
// Styling can also be toggled at runtime with [Disable] and [Enable], and the
// current state can be queried with [Enabled].
//
// # Colors
//
// Eight foreground colors are provided:
//
//	fmt.Println(style.Red("error"))
//	fmt.Println(style.Green("ok"))
//	fmt.Println(style.Yellow("warning"))
//	fmt.Println(style.Blue("info"))
//	fmt.Println(style.Magenta("debug"))
//	fmt.Println(style.Cyan("trace"))
//	fmt.Println(style.White("output"))
//	fmt.Println(style.Gray("muted"))
//
// # Text Styles
//
// Three text styles are provided:
//
//	fmt.Println(style.Bold("heading"))
//	fmt.Println(style.Dim("secondary"))
//	fmt.Println(style.Underline("link"))
//
// # Composition
//
// Functions compose naturally by nesting calls. The extra reset code produced
// by nesting is harmless — terminals stop at the first reset:
//
//	label := style.Bold(style.Red("FAIL"))
//	fmt.Println(label)
//
// # Stripping ANSI Codes
//
// [Reset] strips all ANSI SGR escape sequences from a string, useful for
// computing display widths or writing to non-terminal outputs:
//
//	plain := style.Reset(style.Bold(style.Red("text")))
//	// plain == "text"
//
// # NO_COLOR Support
//
// When the NO_COLOR environment variable is set before the program starts,
// the package initialises with styling disabled. Subsequent calls to [Enable]
// override this for the duration of the process.
package style

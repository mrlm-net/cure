// Package env provides runtime environment detection with a cached singleton.
//
// The package detects the current operating system, architecture, shell, and
// tool availability at runtime. The [Detect] function returns a cached
// [Environment] value populated on first call; subsequent calls return the
// same cached data without re-executing any subprocesses.
//
// # Usage
//
//	env := env.Detect()
//	fmt.Println(env.OS)         // e.g. "darwin"
//	fmt.Println(env.Arch)       // e.g. "arm64"
//	fmt.Println(env.GoVersion)  // e.g. "go1.25.0"
//	fmt.Println(env.GitVersion) // e.g. "git version 2.39.0"
//	fmt.Println(env.WorkDir)    // e.g. "/Users/user/project"
//
// # Tool availability
//
// Use [HasTool] to check whether an external program is available on PATH:
//
//	if env.HasTool("docker") {
//	    // docker is available
//	}
//
// # Git repository detection
//
// Use [IsGitRepo] to check whether the current working directory is inside a
// git repository. The function walks up the directory tree looking for a .git
// directory:
//
//	if env.IsGitRepo() {
//	    // cwd is inside a git repository
//	}
package env

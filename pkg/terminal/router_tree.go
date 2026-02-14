package terminal

import (
	"fmt"
	"sort"
	"strings"
)

// node is an internal radix tree node for command storage.
// The tree compresses common prefixes into shared edges, providing O(k)
// lookup where k is the command name length. This structure enables future
// prefix matching, suggestions, and parameterized routing.
type node struct {
	// prefix is the edge label leading to this node.
	prefix string

	// command is stored at this node if isEnd is true.
	command Command

	// isEnd marks this node as terminal (stores a command).
	isEnd bool

	// children maps the first byte of each child's prefix to the child node.
	children map[byte]*node
}

// insert adds a command to the radix tree under the given name.
// Panics if a command with the same name already exists.
func (n *node) insert(name string, cmd Command) {
	if name == "" {
		if n.isEnd {
			panic(fmt.Sprintf("terminal: duplicate command: %q", cmd.Name()))
		}
		n.command = cmd
		n.isEnd = true
		return
	}

	firstByte := name[0]
	child, exists := n.children[firstByte]

	if !exists {
		if n.children == nil {
			n.children = make(map[byte]*node)
		}
		n.children[firstByte] = &node{
			prefix:  name,
			command: cmd,
			isEnd:   true,
		}
		return
	}

	// Find common prefix length
	commonLen := commonPrefixLen(name, child.prefix)

	if commonLen == len(child.prefix) {
		// Child prefix fully consumed — recurse into child
		child.insert(name[commonLen:], cmd)
		return
	}

	// Split: create intermediate node at the common prefix boundary.
	//
	// Example: inserting "test" when "testing" exists:
	//   Before: root -> [testing]
	//   After:  root -> [test] -> [ing]
	splitNode := &node{
		prefix:   child.prefix[:commonLen],
		children: make(map[byte]*node),
	}

	// Move old child below the split with its remaining suffix
	oldSuffix := child.prefix[commonLen:]
	child.prefix = oldSuffix
	splitNode.children[oldSuffix[0]] = child

	// Insert the new command
	newSuffix := name[commonLen:]
	if newSuffix == "" {
		splitNode.command = cmd
		splitNode.isEnd = true
	} else {
		splitNode.children[newSuffix[0]] = &node{
			prefix:  newSuffix,
			command: cmd,
			isEnd:   true,
		}
	}

	n.children[firstByte] = splitNode
}

// search looks up a command by exact name match.
// Returns the command and true if found, nil and false otherwise.
func (n *node) search(name string) (Command, bool) {
	if name == "" {
		if n.isEnd {
			return n.command, true
		}
		return nil, false
	}

	child, exists := n.children[name[0]]
	if !exists {
		return nil, false
	}

	if !strings.HasPrefix(name, child.prefix) {
		return nil, false
	}

	return child.search(name[len(child.prefix):])
}

// collectCommands recursively gathers all commands stored in the tree.
// Returns commands in no guaranteed order.
func (n *node) collectCommands() []Command {
	var commands []Command
	if n.isEnd && n.command != nil {
		commands = append(commands, n.command)
	}
	for _, child := range n.children {
		commands = append(commands, child.collectCommands()...)
	}
	return commands
}

// findSimilar finds commands whose names are similar to the given string,
// using Levenshtein distance. Returns up to maxResults matches, ordered
// by distance (closest first). Only returns commands within a distance
// threshold of max(2, len(name)/3).
func (n *node) findSimilar(name string, maxResults int) []Command {
	if name == "" || maxResults <= 0 {
		return nil
	}

	threshold := len(name) / 3
	if threshold < 2 {
		threshold = 2
	}

	type scored struct {
		cmd  Command
		dist int
	}

	all := n.collectCommands()
	var matches []scored
	for _, cmd := range all {
		d := levenshtein(name, cmd.Name())
		if d <= threshold {
			matches = append(matches, scored{cmd: cmd, dist: d})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].dist != matches[j].dist {
			return matches[i].dist < matches[j].dist
		}
		return matches[i].cmd.Name() < matches[j].cmd.Name()
	})

	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	if len(matches) == 0 {
		return nil
	}

	result := make([]Command, len(matches))
	for i, m := range matches {
		result[i] = m.cmd
	}
	return result
}

// commonPrefixLen returns the length of the common prefix between two strings.
func commonPrefixLen(a, b string) int {
	maxLen := min(len(a), len(b))
	for i := 0; i < maxLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return maxLen
}

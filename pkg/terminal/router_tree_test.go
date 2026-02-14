package terminal

import (
	"fmt"
	"sort"
	"testing"
)

func TestNode_InsertAndSearch(t *testing.T) {
	tests := []struct {
		name     string
		commands []string
		search   string
		wantOk   bool
	}{
		{
			name:     "single command",
			commands: []string{"version"},
			search:   "version",
			wantOk:   true,
		},
		{
			name:     "multiple distinct",
			commands: []string{"version", "help", "generate"},
			search:   "help",
			wantOk:   true,
		},
		{
			name:     "shared prefix both found",
			commands: []string{"test", "testing"},
			search:   "test",
			wantOk:   true,
		},
		{
			name:     "shared prefix longer found",
			commands: []string{"test", "testing"},
			search:   "testing",
			wantOk:   true,
		},
		{
			name:     "not found",
			commands: []string{"version", "help"},
			search:   "unknown",
			wantOk:   false,
		},
		{
			name:     "empty tree",
			commands: []string{},
			search:   "anything",
			wantOk:   false,
		},
		{
			name:     "prefix of existing not found",
			commands: []string{"generate"},
			search:   "gen",
			wantOk:   false,
		},
		{
			name:     "longer than existing not found",
			commands: []string{"gen"},
			search:   "generate",
			wantOk:   false,
		},
		{
			name:     "many shared prefixes",
			commands: []string{"config", "configure", "confirm", "connect"},
			search:   "confirm",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &node{children: make(map[byte]*node)}
			for _, name := range tt.commands {
				root.insert(name, &mockCommand{name: name})
			}

			cmd, ok := root.search(tt.search)
			if ok != tt.wantOk {
				t.Errorf("search(%q) ok = %v, want %v", tt.search, ok, tt.wantOk)
			}
			if ok && cmd.Name() != tt.search {
				t.Errorf("search(%q) name = %q, want %q", tt.search, cmd.Name(), tt.search)
			}
		})
	}
}

func TestNode_InsertDuplicatePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate insert")
		}
	}()

	root := &node{children: make(map[byte]*node)}
	root.insert("version", &mockCommand{name: "version"})
	root.insert("version", &mockCommand{name: "version"})
}

func TestNode_CollectCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands []string
		want     int
	}{
		{"empty", []string{}, 0},
		{"single", []string{"version"}, 1},
		{"multiple", []string{"version", "help", "generate"}, 3},
		{"shared prefixes", []string{"test", "testing", "template"}, 3},
		{"many commands", []string{"a", "b", "c", "d", "e"}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &node{children: make(map[byte]*node)}
			for _, name := range tt.commands {
				root.insert(name, &mockCommand{name: name})
			}

			got := root.collectCommands()
			if len(got) != tt.want {
				t.Errorf("collectCommands() len = %d, want %d", len(got), tt.want)
			}

			// Verify all commands are present
			names := make([]string, len(got))
			for i, cmd := range got {
				names[i] = cmd.Name()
			}
			sort.Strings(names)
			sort.Strings(tt.commands)
			for i, want := range tt.commands {
				if names[i] != want {
					t.Errorf("command[%d] = %q, want %q", i, names[i], want)
				}
			}
		})
	}
}

func TestNode_FindSimilar_Basic(t *testing.T) {
	root := &node{children: make(map[byte]*node)}
	root.insert("help", &mockCommand{name: "help"})

	got := root.findSimilar("hel", 5)
	if len(got) != 1 || got[0].Name() != "help" {
		names := make([]string, len(got))
		for i, c := range got {
			names[i] = c.Name()
		}
		t.Errorf("findSimilar(hel) = %v, want [help]", names)
	}
}

func TestCommonPrefixLen(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"", "abc", 0},
		{"abc", "abc", 3},
		{"abc", "abd", 2},
		{"abc", "xyz", 0},
		{"test", "testing", 4},
		{"hello", "help", 3},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			got := commonPrefixLen(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("commonPrefixLen(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func BenchmarkNode_Insert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := &node{children: make(map[byte]*node)}
		for j := 0; j < 100; j++ {
			root.insert(fmt.Sprintf("command-%d", j), &mockCommand{name: fmt.Sprintf("command-%d", j)})
		}
	}
}

func BenchmarkNode_Search(b *testing.B) {
	root := &node{children: make(map[byte]*node)}
	for i := 0; i < 100; i++ {
		root.insert(fmt.Sprintf("command-%d", i), &mockCommand{name: fmt.Sprintf("command-%d", i)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = root.search("command-50")
	}
}

func BenchmarkNode_Search_SharedPrefix(b *testing.B) {
	root := &node{children: make(map[byte]*node)}
	commands := []string{"config", "configure", "confirm", "connect", "convert", "copy"}
	for _, name := range commands {
		root.insert(name, &mockCommand{name: name})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = root.search("configure")
	}
}
